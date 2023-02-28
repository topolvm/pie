package controllers

import (
	"context"
	"strings"
	"time"

	"github.com/topolvm/pie/constants"
	"github.com/topolvm/pie/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client client.Client

	storageClasses []string
	namespace      string
	startTime      time.Time
	po             *provisionObserver
}

func NewPodReconciler(
	client client.Client,
	createProbeThreshold time.Duration,
	exporter controller.MetricsExporter,
	storageClasses []string,
	namespace string,
) *PodReconciler {
	return &PodReconciler{
		client:         client,
		storageClasses: storageClasses,
		namespace:      namespace,
		startTime:      time.Now(),
		po:             newProvisionObserver(client, namespace, exporter, createProbeThreshold),
	}
}

func isProbePodName(s string) bool {
	return strings.HasPrefix(s, constants.ProbeNamePrefix)
}

//+kubebuilder:rbac:namespace=default,groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:namespace=default,groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:namespace=default,groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	if !isProbePodName(req.Name) {
		return ctrl.Result{}, nil
	}

	var pod corev1.Pod
	err := r.client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.po.deleteEventTime(pod.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.po.setPodRegisteredTime(pod.Name, pod.CreationTimestamp.Time)

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != constants.ProbeContainerName {
			continue
		}
		if status.State.Running != nil {
			r.po.setPodStartedTime(pod.Name, status.State.Running.StartedAt.Time)
		} else if status.State.Terminated != nil {
			r.po.setPodStartedTime(pod.Name, status.State.Terminated.StartedAt.Time)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.Add(r.po)
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
