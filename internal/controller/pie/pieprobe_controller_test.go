package pie

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	piev1alpha1 "github.com/topolvm/pie/api/pie/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func prepareObjects(ctx context.Context) error {
	_ = log.FromContext(ctx)

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sc",
		},
		Provisioner: "sc-provisioner",
	}
	_, err := ctrl.CreateOrUpdate(ctx, k8sClient, storageClass, func() error { return nil })
	if err != nil {
		return err
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "192.168.0.1",
			Labels: map[string]string{"key1": "value1"},
		},
	}
	_, err = ctrl.CreateOrUpdate(ctx, k8sClient, node, func() error { return nil })
	if err != nil {
		return err
	}

	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "192.168.0.2",
			Labels: map[string]string{"key1": "value1"},
		},
	}
	_, err = ctrl.CreateOrUpdate(ctx, k8sClient, node2, func() error { return nil })
	if err != nil {
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      hostname,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "controller",
					Image: "dummy.image",
				},
			},
		},
	}
	_, err = ctrl.CreateOrUpdate(ctx, k8sClient, pod, func() error { return nil })
	if err != nil {
		return err
	}

	return nil
}

var _ = Describe("PieProbe controller", func() {
	ctx := context.Background()
	var stopFunc func()

	nodeSelector := corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{
			{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "key1",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"value1"},
					},
				},
			},
		},
	}

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:  scheme,
			Metrics: metricsserver.Options{BindAddress: "0"},
		})
		Expect(err).NotTo(HaveOccurred())

		err = prepareObjects(ctx)
		Expect(err).NotTo(HaveOccurred())

		pieProbeReconciler := NewPieProbeController(
			k8sClient,
		)
		err = pieProbeReconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		pieProbe := &piev1alpha1.PieProbe{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "pie-probe-sc",
			},
			Spec: piev1alpha1.PieProbeSpec{
				MonitoringStorageClass: "sc",
				ContainerImage:         "dummy.image",
				ControllerUrl:          "http://localhost:8082",
				NodeSelector:           nodeSelector,
				ProbePeriod:            1,
			},
		}
		_, err = ctrl.CreateOrUpdate(ctx, k8sClient, pieProbe, func() error { return nil })
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should reject to edit monitoringStorageClass", func() {
		By("trying to edit monitoringStorageClass")
		var pieProbe piev1alpha1.PieProbe
		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "pie-probe-sc", Namespace: "default"}, &pieProbe)
			g.Expect(err).ShouldNot(HaveOccurred())
		}).Should(Succeed())

		pieProbe.Spec.MonitoringStorageClass = "sc2"
		err := k8sClient.Update(ctx, &pieProbe)
		Expect(err).Should(HaveOccurred())
	})

	It("should attach ownerReferences for the CronJobs and PVCs to the PieProbe resource", func() {
		tru := true

		var pieProbe piev1alpha1.PieProbe
		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "pie-probe-sc", Namespace: "default"}, &pieProbe)
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())

		By("confirming that the CronJob and PVC have correct ownerReferences")
		Eventually(func(g Gomega) {
			var cronJobList batchv1.CronJobList
			err := k8sClient.List(ctx, &cronJobList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(cronJobList.Items)).Should(Equal(3))
			for _, cronJob := range cronJobList.Items {
				g.Expect(cronJob.OwnerReferences).Should(Equal([]metav1.OwnerReference{{
					APIVersion:         "pie.topolvm.io/v1alpha1",
					Kind:               "PieProbe",
					Name:               "pie-probe-sc",
					UID:                pieProbe.GetUID(),
					Controller:         &tru,
					BlockOwnerDeletion: &tru,
				}}))
			}
		}).Should(Succeed())

		Eventually(func(g Gomega) {
			var pvcList corev1.PersistentVolumeClaimList
			err := k8sClient.List(ctx, &pvcList, client.MatchingLabels(map[string]string{
				"storage-class": "sc",
				"node":          "192.168.0.1",
			}))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(pvcList.Items)).Should(Equal(1))
			g.Expect(pvcList.Items[0].OwnerReferences).Should(Equal([]metav1.OwnerReference{{
				APIVersion:         "pie.topolvm.io/v1alpha1",
				Kind:               "PieProbe",
				Name:               "pie-probe-sc",
				UID:                pieProbe.GetUID(),
				Controller:         &tru,
				BlockOwnerDeletion: &tru,
			}}))
		}).Should(Succeed())
	})

	It("should set correct nodeSelector for provision-probe CronJob", func() {
		By("confirming that the CronJob has correct nodeSelector")
		Eventually(func(g Gomega) {
			var cronJobList batchv1.CronJobList
			selector, err := labels.Parse("storage-class = sc, !node")
			g.Expect(err).NotTo(HaveOccurred())
			err = k8sClient.List(ctx, &cronJobList, &client.ListOptions{
				LabelSelector: selector,
			})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(cronJobList.Items)).Should(Equal(1))
			g.Expect(*cronJobList.Items[0].Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).Should(Equal(nodeSelector))
		}).Should(Succeed())
	})

	It("should delete CronJob and PVC successfully when a node is deleted", func() {
		By("confirming that the CronJob and PVC were created")
		var node corev1.Node
		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "192.168.0.1"}, &node)
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())

		Eventually(func(g Gomega) {
			var cronJobList batchv1.CronJobList
			err := k8sClient.List(ctx, &cronJobList, client.MatchingLabels(map[string]string{
				"storage-class": "sc",
				"node":          "192.168.0.1",
			}))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cronJobList.Items).ShouldNot(BeEmpty())
		}).Should(Succeed())

		Eventually(func(g Gomega) {
			var pvcList corev1.PersistentVolumeClaimList
			err := k8sClient.List(ctx, &pvcList, client.MatchingLabels(map[string]string{
				"storage-class": "sc",
				"node":          "192.168.0.1",
			}))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvcList.Items).ShouldNot(BeEmpty())
		}).Should(Succeed())

		By("confirming that the CronJob and PVC is deleted successfully")
		err := k8sClient.Delete(ctx, &node)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "192.168.0.1"}, &node)
			g.Expect(apierrors.IsNotFound(err)).Should(BeTrue())

			var cronJobList batchv1.CronJobList
			err = k8sClient.List(ctx, &cronJobList, client.MatchingLabels(map[string]string{
				"storage-class": "sc",
				"node":          "192.168.0.1",
			}))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cronJobList.Items).Should(BeEmpty())

			// Check the DeletionTimestamp of the PVC is set.
			// Note that the PVC is not deleted because the finalizer won't be removed in envtest.
			var pvcList corev1.PersistentVolumeClaimList
			err = k8sClient.List(ctx, &pvcList, client.MatchingLabels(map[string]string{
				"storage-class": "sc",
				"node":          "192.168.0.1",
			}))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(pvcList.Items)).Should(Equal(1))
			g.Expect(pvcList.Items[0].DeletionTimestamp).NotTo(BeNil())
		}).Should(Succeed())

		By("confirming that other CronJob and PVC are not deleted")
		Eventually(func(g Gomega) {
			var cronJobList batchv1.CronJobList
			err = k8sClient.List(ctx, &cronJobList, client.MatchingLabels(map[string]string{
				"storage-class": "sc",
				"node":          "192.168.0.2",
			}))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cronJobList.Items).ShouldNot(BeEmpty())

			// Check the DeletionTimestamp of the PVC is not set.
			var pvcList corev1.PersistentVolumeClaimList
			err = k8sClient.List(ctx, &pvcList, client.MatchingLabels(map[string]string{
				"storage-class": "sc",
				"node":          "192.168.0.2",
			}))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(pvcList.Items)).Should(Equal(1))
			g.Expect(pvcList.Items[0].DeletionTimestamp).Should(BeNil())
		}).Should(Succeed())
	})
})
