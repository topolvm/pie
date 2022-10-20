package controllers

import (
	"context"

	"github.com/topolvm/pie/constants"
	batchv1 "k8s.io/api/batch/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// StorageClassReconciler reconciles a StorageClass object
type StorageClassReconciler struct {
	client                   client.Client
	namespace                string
	monitoringStorageClasses map[string]struct{}
}

func NewStorageClassReconciler(
	client client.Client,
	namespace string,
	monitoringStorageClasses []string,
) *StorageClassReconciler {
	storageClass := make(map[string]struct{})
	for _, sc := range monitoringStorageClasses {
		storageClass[sc] = struct{}{}
	}
	return &StorageClassReconciler{
		client:                   client,
		namespace:                namespace,
		monitoringStorageClasses: storageClass,
	}
}

//+kubebuilder:rbac:namespace=default,groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *StorageClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var storageClass storagev1.StorageClass
	err := r.client.Get(ctx, client.ObjectKey{Name: req.Name}, &storageClass)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !storageClass.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&storageClass, constants.StorageClassFinalizerName) {
			label := map[string]string{
				"storage-class": req.Name,
			}
			err := r.client.DeleteAllOf(ctx, &batchv1.CronJob{}, client.InNamespace(r.namespace), client.MatchingLabels(label))
			if err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&storageClass, constants.StorageClassFinalizerName)
			err = r.client.Update(ctx, &storageClass)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&storageClass, constants.StorageClassFinalizerName) {
		controllerutil.AddFinalizer(&storageClass, constants.StorageClassFinalizerName)
		err = r.client.Update(ctx, &storageClass)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StorageClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.NewPredicateFuncs(func(object client.Object) bool {
		_, ok := r.monitoringStorageClasses[object.GetName()]
		return ok
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1.StorageClass{}).
		WithEventFilter(pred).
		Complete(r)
}
