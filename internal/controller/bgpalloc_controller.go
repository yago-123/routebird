package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type BGPAllocReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *BGPAllocReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *BGPAllocReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				svc, ok := e.Object.(*corev1.Service)
				return ok && svc.Spec.Type == corev1.ServiceTypeLoadBalancer
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				newSvc, ok := e.ObjectNew.(*corev1.Service)
				return ok && newSvc.Spec.Type == corev1.ServiceTypeLoadBalancer
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				svc, ok := e.Object.(*corev1.Service)
				return ok && svc.Spec.Type == corev1.ServiceTypeLoadBalancer
			},
		}).
		Complete(r)
}
