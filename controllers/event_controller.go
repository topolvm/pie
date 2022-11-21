package controllers

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/topolvm/pie/constants"
	"github.com/topolvm/pie/controller"
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
	client         client.Client
	storageClasses []string
	namespace      string
	startTime      time.Time
	po             *provisionObserver
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
		client:         client,
		storageClasses: storageClasses,
		namespace:      namespace,
		startTime:      time.Now(),
		po:             newProvisionObserver(client, namespace, exporter, createProbeThreshold, eventTTL),
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
			r.po.setFirstPodEventTime(podName, eventTime.Time)

		}
		if event.Source.Component == "kubelet" && event.Reason == "Created" && eventTime.Time.After(r.startTime) {
			r.po.setPodCreatedEventTime(podName, eventTime.Time)
		}
	}
	return ctrl.Result{}, nil
}

type provisionObserver struct {
	client                client.Client
	namespace             string
	exporter              controller.MetricsExporter
	eventTTL              time.Duration
	createProbeThreshold  time.Duration
	firstPodEventTime     map[string]time.Time
	podCreatedEventTime   map[string]time.Time
	muFirstPodEventTime   sync.Mutex
	muPodCreatedEventTime sync.Mutex
}

func newProvisionObserver(
	client client.Client,
	namespace string,
	exporter controller.MetricsExporter,
	createProbeThreshold time.Duration,
	eventTTL time.Duration,
) *provisionObserver {
	return &provisionObserver{
		client:               client,
		namespace:            namespace,
		exporter:             exporter,
		eventTTL:             eventTTL,
		createProbeThreshold: createProbeThreshold,
		firstPodEventTime:    make(map[string]time.Time),
		podCreatedEventTime:  make(map[string]time.Time),
	}
}

func (p *provisionObserver) setFirstPodEventTime(podName string, eventTime time.Time) {
	p.muFirstPodEventTime.Lock()
	defer p.muFirstPodEventTime.Unlock()

	if t, ok := p.firstPodEventTime[podName]; ok {
		if eventTime.Before(t) {
			p.firstPodEventTime[podName] = eventTime
		}
	} else {
		p.firstPodEventTime[podName] = eventTime
	}
}

func (p *provisionObserver) setPodCreatedEventTime(podName string, eventTime time.Time) {
	p.muPodCreatedEventTime.Lock()
	defer p.muPodCreatedEventTime.Unlock()

	p.podCreatedEventTime[podName] = eventTime
}

func (p *provisionObserver) getNodeNameAndStorageClass(ctx context.Context, podName string) (string, string, error) {
	var pod corev1.Pod
	err := p.client.Get(ctx, client.ObjectKey{Namespace: p.namespace, Name: podName}, &pod)
	if err != nil {
		return "", "", err
	}

	return pod.GetLabels()[constants.ProbeNodeLabelKey], pod.GetLabels()[constants.ProbeStorageClassLabelKey], nil
}

func (p *provisionObserver) deletePod(ctx context.Context, podName string) error {
	var podForDelete corev1.Pod
	err := p.client.Get(ctx, client.ObjectKey{Namespace: p.namespace, Name: podName}, &podForDelete)
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
	err = p.client.Delete(ctx, &podForDelete, &client.DeleteOptions{
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

func (p *provisionObserver) Start(ctx context.Context) error {
	countedFlag := make(map[string]bool)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}

		func() {
			p.muFirstPodEventTime.Lock()
			defer p.muFirstPodEventTime.Unlock()
			p.muPodCreatedEventTime.Lock()
			defer p.muPodCreatedEventTime.Unlock()

			for podName, firstTime := range p.firstPodEventTime {
				if countedFlag[podName] {
					continue
				}
				nodeName, storageClass, err := p.getNodeNameAndStorageClass(ctx, podName)
				if err != nil {
					if !apierrors.IsNotFound(err) {
						eventCtrlLogger.Error(err, "failed to get node and storage class related to the pod", "pod", podName)
					}
					continue
				}
				t, ok := p.podCreatedEventTime[podName]
				if ok {
					countedFlag[podName] = true
					if t.Sub(firstTime) >= p.createProbeThreshold {
						p.exporter.IncrementCreateProbeSlowCount(nodeName, storageClass)
						err := p.deletePod(ctx, podName)
						if err != nil {
							continue
						}
					} else {
						p.exporter.IncrementCreateProbeFastCount(nodeName, storageClass)
					}
				} else {
					if time.Since(firstTime) >= p.createProbeThreshold {
						countedFlag[podName] = true
						p.exporter.IncrementCreateProbeSlowCount(nodeName, storageClass)
						err := p.deletePod(ctx, podName)
						if err != nil {
							continue
						}
					}
				}
				if firstTime.Before(time.Now().Add(-p.eventTTL)) {
					delete(p.firstPodEventTime, podName)
					delete(countedFlag, podName)
					delete(p.podCreatedEventTime, podName)
				}
			}
		}()
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.Add(r.po)
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
