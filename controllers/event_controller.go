package controllers

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/topolvm/csi-driver-availability-monitor/constants"
	"github.com/topolvm/csi-driver-availability-monitor/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	eventCtrlLogger = ctrl.Log.WithName("event-reconciler")
)

// EventReconciler reconciles a Event object
type EventReconciler struct {
	client                client.Client
	firstPodEventTime     map[string]time.Time
	podCreatedEventTime   map[string]time.Time
	createProbeThreshold  time.Duration
	exporter              controller.MetricsExporter
	storageClasses        []string
	namespace             string
	startTime             time.Time
	eventTTL              time.Duration
	muFirstPodEventTime   sync.Mutex
	muPodCreatedEventTime sync.Mutex
}

func NewEventReconciler(
	client client.Client,
	createProbeThreshold time.Duration,
	exporter controller.MetricsExporter,
	storageClasses []string,
	namespace string,
	eventTTL time.Duration,
) *EventReconciler {
	return &EventReconciler{
		client:               client,
		firstPodEventTime:    make(map[string]time.Time),
		podCreatedEventTime:  make(map[string]time.Time),
		createProbeThreshold: createProbeThreshold,
		exporter:             exporter,
		storageClasses:       storageClasses,
		namespace:            namespace,
		startTime:            time.Now(),
		eventTTL:             eventTTL,
	}
}

func isProbePodName(s string) bool {
	return strings.HasPrefix(s, constants.ProbePodNamePrefix)
}

//+kubebuilder:rbac:namespace=default,groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:namespace=default,groups=core,resources=events/status,verbs=get;update;patch
//+kubebuilder:rbac:namespace=default,groups=core,resources=events/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *EventReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var event corev1.Event
	err := r.client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, &event)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if event.InvolvedObject.Kind == "Pod" && isProbePodName(event.InvolvedObject.Name) {
		podName := event.InvolvedObject.Name
		// Use eventTime preferentially because event.FirstTimestamp is deprecated at Kubernetes 1.25.
		var eventTime metav1.MicroTime
		if !event.EventTime.IsZero() {
			eventTime = event.EventTime
		} else {
			eventTime = metav1.MicroTime(event.FirstTimestamp)
		}
		if !eventTime.IsZero() && eventTime.Time.After(r.startTime) {
			func() {
				r.muFirstPodEventTime.Lock()
				defer r.muFirstPodEventTime.Unlock()
				if t, ok := r.firstPodEventTime[podName]; ok {
					if eventTime.Time.Before(t) {
						r.firstPodEventTime[podName] = eventTime.Time
					}
				} else {
					r.firstPodEventTime[podName] = eventTime.Time
				}
			}()
		}
		if event.Source.Component == "kubelet" && event.Reason == "Created" && eventTime.Time.After(r.startTime) {
			r.muPodCreatedEventTime.Lock()
			r.podCreatedEventTime[podName] = eventTime.Time
			r.muPodCreatedEventTime.Unlock()
		}
	}
	return ctrl.Result{}, nil
}

func (r *EventReconciler) getNodeNameAndStorageClass(ctx context.Context, podName string, storageClasses []string) (string, string, error) {
	var pod corev1.Pod
	err := r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: podName}, &pod)
	if err != nil {
		return "", "", err
	}

	return pod.GetLabels()[constants.ProbeNodeLabelKey], pod.GetLabels()[constants.ProbeStorageClassLabelKey], nil
}

func (r *EventReconciler) deletePod(ctx context.Context, podName string) error {
	var podForDelete corev1.Pod
	err := r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: podName}, &podForDelete)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		eventCtrlLogger.Error(err, "failed to get pod", "pod", podName)
		return err
	}
	uid := podForDelete.GetUID()
	resourceVersion := podForDelete.GetResourceVersion()
	cond := metav1.Preconditions{
		UID:             &uid,
		ResourceVersion: &resourceVersion,
	}
	err = r.client.Delete(ctx, &podForDelete, &client.DeleteOptions{
		Preconditions: &cond,
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		eventCtrlLogger.Error(err, "failed to delete pod", "pod", podName)
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventReconciler) SetupWithManager(mgr ctrl.Manager) error {
	go func() {
		ctx := context.Background()
		countedFlag := make(map[string]bool)
		for range time.Tick(time.Second) {
			func() {
				r.muFirstPodEventTime.Lock()
				defer r.muFirstPodEventTime.Unlock()
				r.muPodCreatedEventTime.Lock()
				defer r.muPodCreatedEventTime.Unlock()
				for podName, firstTime := range r.firstPodEventTime {
					if countedFlag[podName] {
						continue
					}
					nodeName, storageClass, err := r.getNodeNameAndStorageClass(ctx, podName, r.storageClasses)
					if err != nil {
						eventCtrlLogger.Error(err, "failed to name of node and storage class related to the pod", "pod", podName)
						continue
					}
					t, ok := r.podCreatedEventTime[podName]
					if ok {
						countedFlag[podName] = true
						if t.Sub(firstTime) >= r.createProbeThreshold {
							r.exporter.IncrementCreateProbeSlowCount(nodeName, storageClass)
							err := r.deletePod(ctx, podName)
							if err != nil {
								continue
							}
						} else {
							r.exporter.IncrementCreateProbeFastCount(nodeName, storageClass)
						}
					} else {
						if time.Since(firstTime) >= r.createProbeThreshold {
							countedFlag[podName] = true
							r.exporter.IncrementCreateProbeSlowCount(nodeName, storageClass)
							err := r.deletePod(ctx, podName)
							if err != nil {
								continue
							}
						}
					}
					if firstTime.Before(time.Now().Add(-r.eventTTL)) {
						delete(r.firstPodEventTime, podName)
						delete(countedFlag, podName)
						delete(r.podCreatedEventTime, podName)
					}
				}
			}()
		}
	}()
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
