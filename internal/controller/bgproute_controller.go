/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bgpv1alphav1 "github.com/yago-123/routebird/api/v1alphav1"
)

// BGPRouteReconciler reconciles a BGPRoute object

// Permissions for managing DaemonSets
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;create;update;list;watch

// Permissions for managing ConfigMaps
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;create;list;watch

type BGPRouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bgp.routebird.dev,resources=bgproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bgp.routebird.dev,resources=bgproutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=bgp.routebird.dev,resources=bgproutes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BGPRoute object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *BGPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var route bgpv1alphav1.BGPRoute
	if err := r.Get(ctx, req.NamespacedName, &route); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	/*
		Create, set up owner reference and create config map for routebird-agent
	*/
	desiredCMap, err := buildAgentConfigMap(route)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Error generating ConfigMap object: %w", err)
	}
	if err = ctrl.SetControllerReference(&route, desiredCMap, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for ConfigMap")
		return ctrl.Result{}, fmt.Errorf("Failed to set owner reference for ConfigMap: %w", err)
	}
	if err = r.reconcileAgentConfigMap(ctx, desiredCMap); err != nil {
		return ctrl.Result{}, err
	}

	/*
		Create, set up owner reference and create service account for routebird-agent
	*/
	saName := route.Spec.Agent.ServiceAccountName
	desiredSAccount := buildAgentServiceAccount(route, saName)
	if err = ctrl.SetControllerReference(&route, &desiredSAccount, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for ServiceAccount", "ServiceAccount.Name", desiredSAccount.Name)
		return ctrl.Result{}, fmt.Errorf("Failed to set owner reference for ServiceAccount: %w", err)
	}
	if err = r.reconcileAgentServiceAccount(ctx, desiredSAccount); err != nil {
		return ctrl.Result{}, err
	}

	/*
		Create, set up owner reference and create daemon set for routebird-agent
	*/
	cfgMapHash := calculateConfigMapHash(desiredCMap.Data)
	desiredDSet := buildAgentDaemonSet(route, desiredCMap.Name, cfgMapHash)
	if err = ctrl.SetControllerReference(&route, &desiredDSet, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for DaemonSet", "DaemonSet.Name", desiredDSet.Name)
		return ctrl.Result{}, fmt.Errorf("Failed to set owner reference for DaemonSet: %w", err)
	}
	if err = r.reconcileAgentDaemonSet(ctx, desiredDSet); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BGPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bgpv1alphav1.BGPRoute{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.DaemonSet{}).
		Named("routebird").
		Complete(r)
}
