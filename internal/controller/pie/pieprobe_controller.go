package pie

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"

	piev1alpha1 "github.com/topolvm/pie/api/pie/v1alpha1"
	"github.com/topolvm/pie/constants"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PieProbeReconciler reconciles a PieProbe object
type PieProbeReconciler struct {
	client client.Client
}

//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes/finalizers,verbs=update
//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:namespace=default,groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PieProbe object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PieProbeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	pieProbe := piev1alpha1.PieProbe{}
	err := r.client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &pieProbe)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var storageClassForGet storagev1.StorageClass
	err = r.client.Get(ctx, client.ObjectKey{Name: pieProbe.Spec.MonitoringStorageClass}, &storageClassForGet)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if storageClassForGet.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// Create a provision-probe CronJob for each sc
	err = r.createOrUpdateJob(
		ctx,
		&pieProbe,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PieProbeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&piev1alpha1.PieProbe{}).
		Complete(r)
}

func makeCronSchedule(pieProbeName string, storageClass string, nodeNamePtr *string, period int) string {
	nodeName := ""
	if nodeNamePtr != nil {
		nodeName = *nodeNamePtr
	}

	h := crc32.NewIEEE()
	h.Write([]byte(pieProbeName))
	h.Write([]byte(storageClass))
	h.Write([]byte(nodeName))

	return fmt.Sprintf("%d-59/%d * * * *", h.Sum32()%uint32(period), period)
}

func addPodFinalizer(spec *corev1.PodTemplateSpec) {
	finalizers := spec.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == constants.PodFinalizerName {
			return
		}
	}
	spec.SetFinalizers(append(finalizers, constants.PodFinalizerName))
}

func (r *PieProbeReconciler) createOrUpdateJob(
	ctx context.Context,
	pieProbe *piev1alpha1.PieProbe,
) error {
	logger := log.FromContext(ctx)
	logger.Info("createOrUpdateJob")
	defer logger.Info("createOrUpdateJob Finished")

	cronjob := &batchv1.CronJob{}
	cronjob.SetNamespace(pieProbe.GetNamespace())
	cronjob.SetName(getCronJobName(nil, pieProbe))

	storageClass := pieProbe.Spec.MonitoringStorageClass

	op, err := ctrl.CreateOrUpdate(ctx, r.client, cronjob, func() error {
		label := map[string]string{
			constants.ProbeStorageClassLabelKey: storageClass,
			constants.ProbePieProbeLabelKey:     pieProbe.GetName(),
		}
		cronjob.SetLabels(label)

		cronjob.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
		cronjob.Spec.Schedule = makeCronSchedule(pieProbe.GetName(), storageClass, nil, pieProbe.Spec.ProbePeriod)

		var successfulJobsHistoryLimit = int32(0)
		cronjob.Spec.SuccessfulJobsHistoryLimit = &successfulJobsHistoryLimit

		// according this doc https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#jobspec-v1-batch,
		// selector is set by the system

		cronjob.Spec.JobTemplate.Spec.Template.SetLabels(label)

		addPodFinalizer(&cronjob.Spec.JobTemplate.Spec.Template)

		if len(cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers) != 1 {
			cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{}}
		}

		volumeName := "genericvol"
		container := &cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
		container.Name = constants.ProbeContainerName
		container.Image = pieProbe.Spec.ContainerImage

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

		container.Args = []string{
			"provision-probe",
		}

		cronjob.Spec.JobTemplate.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &pieProbe.Spec.NodeSelector,
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

		ctrl.SetControllerReference(pieProbe, cronjob, r.client.Scheme())

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create CronJob: %s", getCronJobName(nil, pieProbe))
	}
	if op != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("CronJob successfully created: %s", getCronJobName(nil, pieProbe)))
	}
	return nil
}

// CronJob name should be less than or equal to 52 characters.
// cf. https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/
// One CronJob is created per node and a StorageClass.
// It is desirable that the names of nodes and StorageClasses can be inferred from the names of CronJobs.
// However, if the node and StorageClass names are too long, the CronJob name will not fit in 52 characters.
// So we cut off the node and StorageClass names to an appropriate length and added a hash value at the end
// to balance readability and uniqueness.
func getCronJobName(nodeNamePtr *string, pieProbe *piev1alpha1.PieProbe) string {
	nodeName := ""
	if nodeNamePtr != nil {
		nodeName = *nodeNamePtr
	}

	pieProbeName := pieProbe.Name
	storageClass := pieProbe.Spec.MonitoringStorageClass

	sha1 := sha1.New()
	io.WriteString(sha1, pieProbeName+"\000"+nodeName+"\000"+storageClass)
	hashedName := hex.EncodeToString(sha1.Sum(nil))

	if len(pieProbeName) > 10 {
		pieProbeName = pieProbeName[:10]
	}
	if len(nodeName) > 11 {
		nodeName = nodeName[:11]
	}
	if len(storageClass) > 14 {
		storageClass = storageClass[:14]
	}

	return fmt.Sprintf("%s-%s-%s-%s", constants.ProvisionProbeNamePrefix, pieProbeName, storageClass, hashedName[:6])
}

func NewPieProbeController(
	client client.Client,
) *PieProbeReconciler {
	return &PieProbeReconciler{
		client: client,
	}
}
