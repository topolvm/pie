package pie

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"time"

	piev1alpha1 "github.com/topolvm/pie/api/pie/v1alpha1"
	"github.com/topolvm/pie/constants"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ProvisionProbe = iota
	MountProbe
)

// PieProbeReconciler reconciles a PieProbe object
type PieProbeReconciler struct {
	client         client.Client
	containerImage string
	controllerUrl  string
}

//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes/finalizers,verbs=update
//+kubebuilder:rbac:groups=pie.topolvm.io,resources=pieprobes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:namespace=default,groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:namespace=default,groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch

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

	if time.Duration(pieProbe.Spec.ProbePeriod)*time.Minute <= pieProbe.Spec.ProbeThreshold.Duration {
		return ctrl.Result{}, errors.New("probe period should be larger than probe threshold")
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
		ProvisionProbe,
		&pieProbe,
		nil,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get a node list and create a PVC and a mount-probe CronJob for each node and sc.
	nodeSelector, err := nodeaffinity.NewNodeSelector(&pieProbe.Spec.NodeSelector)
	if err != nil {
		return ctrl.Result{}, err
	}
	allNodeList := corev1.NodeList{}
	r.client.List(ctx, &allNodeList)
	availableNodeList := []corev1.Node{}
	for _, node := range allNodeList.Items {
		if !nodeSelector.Match(&node) {
			continue
		}
		availableNodeList = append(availableNodeList, node)

		err = r.createOrUpdatePVC(
			ctx,
			node.Name,
			&pieProbe,
		)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.createOrUpdateJob(
			ctx,
			MountProbe,
			&pieProbe,
			&node.Name,
		)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	availableNodes := map[string]struct{}{}
	for _, node := range availableNodeList {
		availableNodes[node.GetName()] = struct{}{}
	}

	// Delete unnecessary mount-probe CronJobs
	cronJobList := batchv1.CronJobList{}
	err = r.client.List(ctx, &cronJobList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			constants.ProbePieProbeLabelKey: pieProbe.GetName(),
		}),
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, cronJob := range cronJobList.Items {
		nodeName := cronJob.GetLabels()[constants.ProbeNodeLabelKey]
		if nodeName == "" {
			continue
		}
		if _, ok := availableNodes[nodeName]; ok {
			continue
		}
		err := r.deleteCronJob(ctx, &cronJob)
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
	}

	// Delete unnecessary PVCs
	pvcList := corev1.PersistentVolumeClaimList{}
	err = r.client.List(ctx, &pvcList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			constants.ProbePieProbeLabelKey: pieProbe.GetName(),
		}),
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, pvc := range pvcList.Items {
		nodeName := pvc.GetLabels()[constants.ProbeNodeLabelKey]
		if _, ok := availableNodes[nodeName]; ok {
			continue
		}
		err = r.deletePVC(ctx, &pvc)
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PieProbeReconciler) deletePVC(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	logger := log.FromContext(ctx)

	uid := pvc.GetUID()
	resourceVersion := pvc.GetResourceVersion()
	cond := metav1.Preconditions{
		UID:             &uid,
		ResourceVersion: &resourceVersion,
	}
	err := r.client.Delete(ctx, pvc, &client.DeleteOptions{
		Preconditions: &cond,
	})
	if err != nil {
		logger.Error(err, "failed to delete pvc", "pvcName", pvc.GetName())
		return err
	}
	return nil
}

func (r *PieProbeReconciler) deleteCronJob(ctx context.Context, cronJobForDelete *batchv1.CronJob) error {
	logger := log.FromContext(ctx)

	uid := cronJobForDelete.GetUID()
	resourceVersion := cronJobForDelete.GetResourceVersion()
	cond := metav1.Preconditions{
		UID:             &uid,
		ResourceVersion: &resourceVersion,
	}
	err := r.client.Delete(ctx, cronJobForDelete, &client.DeleteOptions{
		Preconditions: &cond,
	})
	if err != nil {
		logger.Error(err, "failed to delete cronJob", "cronJob", cronJobForDelete.GetName())
		return err
	}
	return nil
}

func (r *PieProbeReconciler) findPieProbesForNode(ctx context.Context, node client.Object) []reconcile.Request {
	pieProbeList := piev1alpha1.PieProbeList{}
	err := r.client.List(ctx, &pieProbeList)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := []reconcile.Request{}
	for _, item := range pieProbeList.Items {
		nodeSelector, err := nodeaffinity.NewNodeSelector(&item.Spec.NodeSelector)
		if err != nil {
			continue
		}
		if nodeSelector.Match(node.(*corev1.Node)) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *PieProbeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&piev1alpha1.PieProbe{}).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.findPieProbesForNode),
		).
		Complete(r)
}

func getPVCName(nodeName string, pieProbe *piev1alpha1.PieProbe) string {
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
	return fmt.Sprintf("%s-%s-%s-%s-%s", constants.PVCNamePrefix, pieProbeName, nodeName, storageClass, hashedName[:6])
}

func (r *PieProbeReconciler) createOrUpdatePVC(
	ctx context.Context,
	nodeName string,
	pieProbe *piev1alpha1.PieProbe,
) error {
	logger := log.FromContext(ctx)

	pvcName := getPVCName(nodeName, pieProbe)
	storageClass := pieProbe.Spec.MonitoringStorageClass

	pvc := &corev1.PersistentVolumeClaim{}
	pvc.SetNamespace(pieProbe.GetNamespace())
	pvc.SetName(pvcName)

	op, err := ctrl.CreateOrUpdate(ctx, r.client, pvc, func() error {
		label := map[string]string{
			constants.ProbeStorageClassLabelKey: storageClass,
			constants.ProbeNodeLabelKey:         nodeName,
			constants.ProbePieProbeLabelKey:     pieProbe.GetName(),
		}
		pvc.SetLabels(label)

		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		pvc.Spec.StorageClassName = &storageClass
		pvc.Spec.Resources = corev1.VolumeResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: *resource.NewQuantity(
					100*1024*1024, resource.BinarySI),
			},
		}

		ctrl.SetControllerReference(pieProbe, pvc, r.client.Scheme())

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create PVC '%s' of storageclass %s: %w", pvcName, storageClass, err)
	}
	if op != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("PVC '%s' successfully created of storageclass %s: %s", pvcName, storageClass, op))
	}

	return nil
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
	kind int,
	pieProbe *piev1alpha1.PieProbe,
	nodeName *string,
) error {
	logger := log.FromContext(ctx)
	logger.Info("createOrUpdateJob")
	defer logger.Info("createOrUpdateJob Finished")

	cronjob := &batchv1.CronJob{}
	cronjob.SetNamespace(pieProbe.GetNamespace())
	cronjob.SetName(getCronJobName(kind, nodeName, pieProbe))

	storageClass := pieProbe.Spec.MonitoringStorageClass

	op, err := ctrl.CreateOrUpdate(ctx, r.client, cronjob, func() error {
		label := map[string]string{
			constants.ProbeStorageClassLabelKey: storageClass,
			constants.ProbePieProbeLabelKey:     pieProbe.GetName(),
		}
		if nodeName != nil {
			label[constants.ProbeNodeLabelKey] = *nodeName
		}
		cronjob.SetLabels(label)

		cronjob.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
		cronjob.Spec.Schedule = makeCronSchedule(pieProbe.GetName(), storageClass, nodeName, pieProbe.Spec.ProbePeriod)

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
		container.Image = r.containerImage

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

		switch kind {
		case ProvisionProbe:
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
									Resources: corev1.VolumeResourceRequirements{
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
		case MountProbe:
			container.VolumeMounts = []corev1.VolumeMount{
				{
					Name:      volumeName,
					MountPath: "/mounted",
				},
			}
			container.Args = []string{
				"probe",
				fmt.Sprintf("--destination-address=%s", r.controllerUrl),
				"--path=/mounted/",
				fmt.Sprintf("--node-name=%s", *nodeName),
				fmt.Sprintf("--storage-class=%s", storageClass),
				fmt.Sprintf("--pie-probe-name=%s", pieProbe.GetName()),
			}
			cronjob.Spec.JobTemplate.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      corev1.LabelHostname,
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{*nodeName},
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
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: getPVCName(*nodeName, pieProbe),
						},
					},
				},
			}
		}

		ctrl.SetControllerReference(pieProbe, cronjob, r.client.Scheme())

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create CronJob: %s", getCronJobName(kind, nodeName, pieProbe))
	}
	if op != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("CronJob successfully created: %s", getCronJobName(kind, nodeName, pieProbe)))
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
func getCronJobName(kind int, nodeNamePtr *string, pieProbe *piev1alpha1.PieProbe) string {
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
	if len(nodeName) > 15 {
		nodeName = nodeName[:15]
	}
	if len(storageClass) > 12 {
		storageClass = storageClass[:12]
	}

	if kind == ProvisionProbe {
		return fmt.Sprintf("%s-%s-%s-%s", constants.ProvisionProbeNamePrefix, pieProbeName, storageClass, hashedName[:6])
	} else { // kind == MountProbe
		return fmt.Sprintf("%s-%s-%s-%s-%s", constants.MountProbeNamePrefix, pieProbeName, nodeName, storageClass, hashedName[:6])
	}
}

func NewPieProbeController(
	client client.Client,
	containerImage string,
	controllerUrl string,
) *PieProbeReconciler {
	return &PieProbeReconciler{
		client:         client,
		containerImage: containerImage,
		controllerUrl:  controllerUrl,
	}
}
