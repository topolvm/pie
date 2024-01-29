package controller

import (
	"context"
	"strings"
	"time"

	piev1alpha1 "github.com/topolvm/pie/api/pie/v1alpha1"
	"github.com/topolvm/pie/constants"
	"github.com/topolvm/pie/metrics"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProbePodReconciler reconciles a Pod object
type ProbePodReconciler struct {
	client client.Client

	startTime time.Time
	po        *provisionObserver2
}

func NewProbePodReconciler(
	client client.Client,
	exporter metrics.MetricsExporter,
) *ProbePodReconciler {
	return &ProbePodReconciler{
		client:    client,
		startTime: time.Now(),
		po:        newProvisionObserver2(client, exporter),
	}
}

//+kubebuilder:rbac:namespace=default,groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ProbePodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	if !strings.HasPrefix(req.Name, constants.ProvisionProbeNamePrefix) {
		return ctrl.Result{}, nil
	}

	var pod corev1.Pod
	err := r.client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(&pod, constants.PodFinalizerName) {
		return ctrl.Result{}, nil
	}

	r.po.setPodRegisteredTime(pod.Namespace, pod.Name, pod.CreationTimestamp.Time)

	pieProbeName := pod.Labels[constants.ProbePieProbeLabelKey]
	var pieProbe piev1alpha1.PieProbe
	err = r.client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: pieProbeName}, &pieProbe)
	if err != nil {
		return ctrl.Result{}, err
	}
	r.po.setPodPieProbeName(pod.Namespace, pod.Name, pieProbeName)

	probeThreshold, err := time.ParseDuration(pieProbe.Spec.ProbeThreshold)
	if err != nil {
		return ctrl.Result{}, err
	}
	r.po.setProbeThreshold(pod.Namespace, pod.Name, probeThreshold)

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != constants.ProbeContainerName {
			continue
		}
		if status.State.Running != nil {
			r.po.setPodStartedTime(pod.Namespace, pod.Name, status.State.Running.StartedAt.Time)
		} else if status.State.Terminated != nil {
			r.po.setPodStartedTime(pod.Namespace, pod.Name, status.State.Terminated.StartedAt.Time)
		}
	}

	if !pod.DeletionTimestamp.IsZero() {
		r.po.deleteEventTime(pod.Namespace, pod.Name)
		controllerutil.RemoveFinalizer(&pod, constants.PodFinalizerName)
		err := r.client.Update(ctx, &pod)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProbePodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.Add(r.po)
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
