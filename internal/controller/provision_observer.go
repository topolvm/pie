package controller

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/topolvm/pie/constants"
	"github.com/topolvm/pie/controller"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = ctrl.Log.WithName("provision-observer")
)

type provisionObserver struct {
	client               client.Client
	namespace            string
	exporter             controller.MetricsExporter
	createProbeThreshold time.Duration
	podRegisteredTime    map[string]time.Time
	podStartedTime       map[string]time.Time
	countedFlag          map[string]struct{}
	// mu protects above maps
	mu          sync.Mutex
	makeCheckCh chan struct{}
	checkDoneCh chan struct{}
}

func newProvisionObserver(
	client client.Client,
	namespace string,
	exporter controller.MetricsExporter,
	createProbeThreshold time.Duration,
) *provisionObserver {
	return &provisionObserver{
		client:               client,
		namespace:            namespace,
		exporter:             exporter,
		createProbeThreshold: createProbeThreshold,
		podRegisteredTime:    make(map[string]time.Time),
		podStartedTime:       make(map[string]time.Time),
		countedFlag:          make(map[string]struct{}),
		makeCheckCh:          make(chan struct{}),
		checkDoneCh:          make(chan struct{}),
	}
}

func (p *provisionObserver) setPodRegisteredTime(podName string, eventTime time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.podRegisteredTime[podName] = eventTime
}

func (p *provisionObserver) setPodStartedTime(podName string, eventTime time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.podStartedTime[podName] = eventTime
}

func (p *provisionObserver) deleteEventTime(podName string) {
	p.makeCheckCh <- struct{}{}
	<-p.checkDoneCh

	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.podRegisteredTime, podName)
	delete(p.podStartedTime, podName)
	delete(p.countedFlag, podName)
}

func (p *provisionObserver) getNodeNameAndStorageClass(ctx context.Context, podName string) (string, string, error) {
	var pod corev1.Pod
	err := p.client.Get(ctx, client.ObjectKey{Namespace: p.namespace, Name: podName}, &pod)
	if err != nil {
		return "", "", err
	}

	return pod.GetLabels()[constants.ProbeNodeLabelKey], pod.GetLabels()[constants.ProbeStorageClassLabelKey], nil
}

func isProbeJob(o metav1.OwnerReference) bool {
	return o.Kind == "Job" && strings.HasPrefix(o.Name, constants.ProbeNamePrefix)
}

func (p *provisionObserver) deleteOwnerJobOfPod(ctx context.Context, podName string) error {
	var pod corev1.Pod
	err := p.client.Get(ctx, client.ObjectKey{Namespace: p.namespace, Name: podName}, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		logger.Error(err, "failed to get pod", "pod", podName)
		return err
	}

	for _, ownerReference := range pod.GetOwnerReferences() {
		if isProbeJob(ownerReference) {
			var job batchv1.Job
			err := p.client.Get(ctx, client.ObjectKey{Namespace: p.namespace, Name: ownerReference.Name}, &job)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				logger.Error(err, "failed to get job", "job", ownerReference.Name)
				return err
			}

			policy := metav1.DeletePropagationBackground
			err = p.client.Delete(ctx, &job, &client.DeleteOptions{PropagationPolicy: &policy})
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				logger.Error(err, "failed to delete job", "job", job.Name)
				return err
			}
		}
	}

	return nil
}

func (p *provisionObserver) check(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for podName, registeredTime := range p.podRegisteredTime {
		if _, ok := p.countedFlag[podName]; ok {
			continue
		}
		nodeName, storageClass, err := p.getNodeNameAndStorageClass(ctx, podName)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "failed to get node and storage class related to the pod", "pod", podName)
			}
			continue
		}
		t, ok := p.podStartedTime[podName]
		if ok {
			p.countedFlag[podName] = struct{}{}
			if t.Sub(registeredTime) >= p.createProbeThreshold {
				p.exporter.IncrementCreateProbeCount(nodeName, storageClass, false)
				err := p.deleteOwnerJobOfPod(ctx, podName)
				if err != nil {
					continue
				}
			} else {
				p.exporter.IncrementCreateProbeCount(nodeName, storageClass, true)
			}
		} else {
			if time.Since(registeredTime) >= p.createProbeThreshold {
				p.countedFlag[podName] = struct{}{}
				p.exporter.IncrementCreateProbeCount(nodeName, storageClass, false)
				err := p.deleteOwnerJobOfPod(ctx, podName)
				if err != nil {
					continue
				}
			}
		}
	}
}

//+kubebuilder:rbac:namespace=default,groups=batch,resources=jobs,verbs=get;list;watch;delete

func (p *provisionObserver) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		needNotify := false
		select {
		case <-p.makeCheckCh:
			needNotify = true
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}

		p.check(ctx)

		if needNotify {
			p.checkDoneCh <- struct{}{}
		}
	}
}
