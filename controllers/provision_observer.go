package controllers

import (
	"context"
	"sync"
	"time"

	"github.com/topolvm/pie/constants"
	"github.com/topolvm/pie/controller"
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
	mu                   sync.Mutex
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

func (p *provisionObserver) deletePod(ctx context.Context, podName string) error {
	var podForDelete corev1.Pod
	err := p.client.Get(ctx, client.ObjectKey{Namespace: p.namespace, Name: podName}, &podForDelete)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		logger.Error(err, "failed to get pod", "pod", podName)
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
		logger.Error(err, "failed to delete pod", "pod", podName)
		return err
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
				p.exporter.IncrementCreateProbeSlowCount(nodeName, storageClass)
				err := p.deletePod(ctx, podName)
				if err != nil {
					continue
				}
			} else {
				p.exporter.IncrementCreateProbeFastCount(nodeName, storageClass)
			}
		} else {
			if time.Since(registeredTime) >= p.createProbeThreshold {
				p.countedFlag[podName] = struct{}{}
				p.exporter.IncrementCreateProbeSlowCount(nodeName, storageClass)
				err := p.deletePod(ctx, podName)
				if err != nil {
					continue
				}
			}
		}
	}
}

func (p *provisionObserver) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}

		p.check(ctx)
	}
}
