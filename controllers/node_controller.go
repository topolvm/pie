package controllers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/topolvm/pie/constants"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	nodeCtrlLogger = ctrl.Log.WithName("node-reconciler")
)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client                   client.Client
	containerImage           string
	namespace                string
	controllerURL            string
	monitoringStorageClasses []string
	nodeSelector             *metav1.LabelSelector
	probePeriod              int
}

func NewNodeReconciler(
	client client.Client,
	containerImage string,
	namespace string,
	controllerURL string,
	monitoringStorageClasses []string,
	nodeSelector *metav1.LabelSelector,
	probePeriod int,
) *NodeReconciler {
	return &NodeReconciler{
		client:                   client,
		containerImage:           containerImage,
		namespace:                namespace,
		controllerURL:            controllerURL,
		monitoringStorageClasses: monitoringStorageClasses,
		nodeSelector:             nodeSelector,
		probePeriod:              probePeriod,
	}
}

//+kubebuilder:rbac:namespace=default,groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:namespace=default,groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=nodes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var node corev1.Node
	err := r.client.Get(ctx, client.ObjectKey{Name: req.Name}, &node)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !node.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&node, constants.NodeFinalizerName) {
			for _, storageClass := range r.monitoringStorageClasses {
				var storageClassForGet storagev1.StorageClass
				err := r.client.Get(ctx, client.ObjectKey{Name: storageClass}, &storageClassForGet)
				if err != nil {
					if apierrors.IsNotFound(err) {
						continue
					}
					return ctrl.Result{}, err
				}
				err = r.deleteCronJob(ctx, getCronJobName(node.Name, storageClass))
				if err != nil {
					if !apierrors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
				}
			}

			controllerutil.RemoveFinalizer(&node, constants.NodeFinalizerName)
			err = r.client.Update(ctx, &node)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	for _, storageClass := range r.monitoringStorageClasses {
		var storageClassForGet storagev1.StorageClass
		err := r.client.Get(ctx, client.ObjectKey{Name: storageClass}, &storageClassForGet)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return ctrl.Result{}, err
		}
		err = r.createOrUpdateJob(ctx, storageClass, node.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !controllerutil.ContainsFinalizer(&node, constants.NodeFinalizerName) {
		controllerutil.AddFinalizer(&node, constants.NodeFinalizerName)
		err = r.client.Update(ctx, &node)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// CronJob name should be less than or equal to 52 characters.
// cf. https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/
// One CronJob is created per node and a StorageClass.
// It is desirable that the names of nodes and StorageClasses can be inferred from the names of CronJobs.
// However, if the node and StorageClass names are too long, the CronJob name will not fit in 52 characters.
// So we cut off the node and StorageClass names to an appropriate length and added a hash value at the end
// to balance readability and uniqueness.
func getCronJobName(nodeName, storageClass string) string {
	sha1 := sha1.New()
	io.WriteString(sha1, nodeName+"\000"+storageClass)
	hashedName := hex.EncodeToString(sha1.Sum(nil))

	if len(nodeName) > 15 {
		nodeName = nodeName[:15]
	}
	if len(storageClass) > 18 {
		storageClass = storageClass[:18]
	}
	return fmt.Sprintf("%s-%s-%s-%s", constants.ProbePodNamePrefix, nodeName, storageClass, hashedName[:6])
}

func (r *NodeReconciler) deleteCronJob(ctx context.Context, cronJobName string) error {
	var cronJobForDelete batchv1.CronJob
	err := r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: cronJobName}, &cronJobForDelete)
	if err != nil {
		nodeCtrlLogger.Error(err, "failed to get cronJob", "cronJob", cronJobName)
		return err
	}
	uid := cronJobForDelete.GetUID()
	resourceVersion := cronJobForDelete.GetResourceVersion()
	cond := metav1.Preconditions{
		UID:             &uid,
		ResourceVersion: &resourceVersion,
	}
	err = r.client.Delete(ctx, &cronJobForDelete, &client.DeleteOptions{
		Preconditions: &cond,
	})
	if err != nil {
		nodeCtrlLogger.Error(err, "failed to delete cronJob", "cronJob", cronJobName)
		return err
	}
	return nil
}

func convertPeriodToCronSchedule(period int) string {
	return fmt.Sprintf("*/%d * * * *", period)
}

func (r *NodeReconciler) createOrUpdateJob(ctx context.Context, storageClass, nodeName string) error {
	nodeCtrlLogger.Info("createOrUpdateJob")
	defer nodeCtrlLogger.Info("createOrUpdateJob Finished")

	cronjob := &batchv1.CronJob{}
	cronjob.SetNamespace(r.namespace)
	cronjob.SetName(getCronJobName(nodeName, storageClass))

	op, err := ctrl.CreateOrUpdate(ctx, r.client, cronjob, func() error {
		label := map[string]string{
			constants.ProbeNodeLabelKey:         nodeName,
			constants.ProbeStorageClassLabelKey: storageClass,
		}
		cronjob.SetLabels(label)

		var controllerPod corev1.Pod
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		err = r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: hostname}, &controllerPod)
		if err != nil {
			return err
		}
		err = controllerutil.SetControllerReference(&controllerPod, cronjob, r.client.Scheme())
		if err != nil {
			return err
		}

		cronjob.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
		cronjob.Spec.Schedule = convertPeriodToCronSchedule(r.probePeriod)
		// according this doc https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#jobspec-v1-batch,
		// selector is set by the system
		// job.Spec.JobTemplate.ObjectMeta.Labels = map[string]string{"name": "pie-probe"}

		cronjob.Spec.JobTemplate.Spec.Template.SetLabels(label)

		if len(cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers) != 1 {
			cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{}}
		}

		volumeName := "genericvol"
		container := &cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
		container.Name = constants.ProbeContainerName
		container.Image = r.containerImage
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: "/mounted",
			},
		}

		container.Args = []string{
			"probe",
			fmt.Sprintf("--destination-address=%s", r.controllerURL),
			"--path=/mounted/",
			fmt.Sprintf("--node-name=%s", nodeName),
			fmt.Sprintf("--storage-class=%s", storageClass),
		}

		var userID int64 = 1001
		var groupID int64 = 1001
		cronjob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  &userID,
			RunAsGroup: &groupID,
			FSGroup:    &groupID,
		}

		cronjob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
		var periodSeconds int64 = 5
		cronjob.Spec.JobTemplate.Spec.Template.Spec.TerminationGracePeriodSeconds = &periodSeconds

		cronjob.Spec.JobTemplate.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      corev1.LabelHostname,
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{nodeName},
								},
							},
						},
					},
				},
			},
		}

		cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					Ephemeral: &corev1.EphemeralVolumeSource{
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								StorageClassName: &storageClass,
								Resources: corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										corev1.ResourceStorage: *resource.NewQuantity(
											100*1024*1024, resource.BinarySI),
									},
								},
							},
						},
					},
				},
			},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create CronJob on node %s of storageclass %s: %w", nodeName, storageClass, err)
	}
	if op != controllerutil.OperationResultNone {
		nodeCtrlLogger.Info(fmt.Sprintf("CronJob successfully created node %s of storageclass %s: %s", nodeName, storageClass, op))
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred, err := predicate.LabelSelectorPredicate(*r.nodeSelector)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(pred).
		Complete(r)
}
