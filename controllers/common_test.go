package controllers

import (
	"context"
	"os"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func prepareObjects(ctx context.Context) error {
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sc",
		},
		Provisioner: "sc-provisioner",
	}
	_, err := ctrl.CreateOrUpdate(ctx, k8sClient, storageClass, func() error { return nil })
	if err != nil {
		return err
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "192.168.0.1",
			Labels: map[string]string{"hoge": "fuga"},
		},
	}
	_, err = ctrl.CreateOrUpdate(ctx, k8sClient, node, func() error { return nil })
	if err != nil {
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      hostname,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "controller",
					Image: "dummy.image",
				},
			},
		},
	}
	_, err = ctrl.CreateOrUpdate(ctx, k8sClient, pod, func() error { return nil })
	if err != nil {
		return err
	}

	return nil
}
