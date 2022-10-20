package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/topolvm/pie/constants"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Node controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
		})
		Expect(err).NotTo(HaveOccurred())

		err = prepareObjects(ctx)
		Expect(err).NotTo(HaveOccurred())

		monitoringStorageClasses := []string{"sc"}
		nodeReconciler := NewNodeReconciler(
			k8sClient,
			"dummy",
			"default",
			"http://localhost:8082",
			monitoringStorageClasses,
			make(map[string]string),
			1,
		)
		err = nodeReconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		storageClassReconciler := NewStorageClassReconciler(k8sClient, "default", monitoringStorageClasses)
		err = storageClassReconciler.SetupWithManager(mgr)
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

	It("should set finalizer and then delete CronJob successfully", func() {
		By("confirming that the finalizer is set")
		var node corev1.Node
		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "192.168.0.1"}, &node)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(controllerutil.ContainsFinalizer(&node, constants.NodeFinalizerName)).Should(BeTrue())
		}).Should(Succeed())

		Eventually(func(g Gomega) {
			var cronJobList batchv1.CronJobList
			err := k8sClient.List(ctx, &cronJobList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cronJobList.Items).ShouldNot(BeEmpty())
		}).Should(Succeed())

		By("confirming that the CronJob is deleted successfully")
		err := k8sClient.Delete(ctx, &node)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "192.168.0.1"}, &node)
			g.Expect(apierrors.IsNotFound(err)).Should(BeTrue())

			var cronJobList batchv1.CronJobList
			err = k8sClient.List(ctx, &cronJobList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cronJobList.Items).Should(BeEmpty())
		}).Should(Succeed())
	})
})
