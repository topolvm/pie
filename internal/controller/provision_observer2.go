package controller

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/topolvm/pie/constants"
	"github.com/topolvm/pie/metrics"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type namespacePod struct {
	namespace string
	podName   string
}

type provisionObserver2 struct {
	client            client.Client
	exporter          metrics.MetricsExporter
	podRegisteredTime map[namespacePod]time.Time
	podStartedTime    map[namespacePod]time.Time
	countedFlag       map[namespacePod]struct{}
	probeThreshold    map[namespacePod]time.Duration
	podPieProbeName   map[namespacePod]string
	// mu protects above maps
	mu          sync.Mutex
	makeCheckCh chan struct{}
	checkDoneCh chan struct{}
}

func newProvisionObserver2(
	client client.Client,
	exporter metrics.MetricsExporter,
) *provisionObserver2 {
	return &provisionObserver2{
		client:            client,
		exporter:          exporter,
		podRegisteredTime: make(map[namespacePod]time.Time),
		podStartedTime:    make(map[namespacePod]time.Time),
		countedFlag:       make(map[namespacePod]struct{}),
		probeThreshold:    make(map[namespacePod]time.Duration),
		podPieProbeName:   make(map[namespacePod]string),
		makeCheckCh:       make(chan struct{}),
		checkDoneCh:       make(chan struct{}),
	}
}

func (p *provisionObserver2) setPodRegisteredTime(namespace, podName string, eventTime time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.podRegisteredTime[namespacePod{namespace, podName}] = eventTime
}

func (p *provisionObserver2) setPodStartedTime(namespace, podName string, eventTime time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.podStartedTime[namespacePod{namespace, podName}] = eventTime
}

func (p *provisionObserver2) setProbeThreshold(namespace, podName string, thr time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.probeThreshold[namespacePod{namespace, podName}] = thr
}

func (p *provisionObserver2) setPodPieProbeName(namespace, podName, pieProbeName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.podPieProbeName[namespacePod{namespace, podName}] = pieProbeName
}

func (p *provisionObserver2) deleteEventTime(namespace, podName string) {
	p.makeCheckCh <- struct{}{}
	<-p.checkDoneCh

	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.podRegisteredTime, namespacePod{namespace, podName})
	delete(p.podStartedTime, namespacePod{namespace, podName})
	delete(p.countedFlag, namespacePod{namespace, podName})
	delete(p.probeThreshold, namespacePod{namespace, podName})
	delete(p.podPieProbeName, namespacePod{namespace, podName})
}

func (p *provisionObserver2) getNodeNameAndStorageClass(ctx context.Context, namespace, podName string) (string, string, error) {
	var pod corev1.Pod
	err := p.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod)
	if err != nil {
		return "", "", err
	}

	return pod.GetLabels()[constants.ProbeNodeLabelKey], pod.GetLabels()[constants.ProbeStorageClassLabelKey], nil
}

func isProbeJob2(o metav1.OwnerReference) bool {
	return o.Kind == "Job" && (strings.HasPrefix(o.Name, constants.MountProbeNamePrefix) || strings.HasPrefix(o.Name, constants.ProvisionProbeNamePrefix))
}

func (p *provisionObserver2) deleteOwnerJobOfPod(ctx context.Context, namespace, podName string) error {
	var pod corev1.Pod
	err := p.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		logger.Error(err, "failed to get pod", "pod", podName)
		return err
	}

	for _, ownerReference := range pod.GetOwnerReferences() {
		if isProbeJob2(ownerReference) {
			var job batchv1.Job
			err := p.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ownerReference.Name}, &job)
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

func (p *provisionObserver2) incrementProbeCount(pieProbeName, podName, nodeName, storageClass string, onTime bool) {
	if strings.HasPrefix(podName, constants.ProvisionProbeNamePrefix) { // ProvisionProbe
		p.exporter.IncrementProvisionProbeCount(pieProbeName, storageClass, onTime)
	} else if strings.HasPrefix(podName, constants.MountProbeNamePrefix) { // MountProbe
		p.exporter.IncrementMountProbeCount(pieProbeName, nodeName, storageClass, onTime)
	}
}

func (p *provisionObserver2) check(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for nsAndPod, registeredTime := range p.podRegisteredTime {
		namespace := nsAndPod.namespace
		podName := nsAndPod.podName
		if _, ok := p.countedFlag[nsAndPod]; ok {
			continue
		}
		nodeName, storageClass, err := p.getNodeNameAndStorageClass(ctx, namespace, podName)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "failed to get node and storage class related to the pod", "pod", podName)
			}
			continue
		}
		probeThreshold := p.probeThreshold[nsAndPod]
		pieProbeName := p.podPieProbeName[nsAndPod]
		t, ok := p.podStartedTime[nsAndPod]
		if ok {
			p.countedFlag[nsAndPod] = struct{}{}
			if t.Sub(registeredTime) >= probeThreshold {
				p.incrementProbeCount(pieProbeName, podName, nodeName, storageClass, false)
				err := p.deleteOwnerJobOfPod(ctx, namespace, podName)
				if err != nil {
					continue
				}
			} else {
				p.incrementProbeCount(pieProbeName, podName, nodeName, storageClass, true)
			}
		} else {
			if time.Since(registeredTime) >= probeThreshold {
				p.countedFlag[nsAndPod] = struct{}{}
				p.incrementProbeCount(pieProbeName, podName, nodeName, storageClass, false)
				err := p.deleteOwnerJobOfPod(ctx, namespace, podName)
				if err != nil {
					continue
				}
			}
		}
	}
}

//+kubebuilder:rbac:namespace=default,groups=batch,resources=jobs,verbs=get;list;watch;delete

func (p *provisionObserver2) Start(ctx context.Context) error {
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
