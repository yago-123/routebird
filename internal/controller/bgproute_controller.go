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
func (r *BGPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var routeCR bgpv1alphav1.BGPRoute
	if err := r.Get(ctx, req.NamespacedName, &routeCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	commonLabels := map[string]string{
		"app":   "routebird-agent",
		"route": routeCR.Name,
	}

	/*
		Create, set up owner reference and create config map for routebird-agent
	*/
	desiredCMap, err := buildAgentConfigMap(routeCR, commonLabels)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Error generating ConfigMap object: %w", err)
	}
	if err = ctrl.SetControllerReference(&routeCR, desiredCMap, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for ConfigMap")
		return ctrl.Result{}, err
	}
	if err = r.reconcileAgentConfigMap(ctx, desiredCMap); err != nil {
		return ctrl.Result{}, err
	}

	/*
		Create, set up owner reference and create service account for routebird-agent
	*/
	desiredSAccount := buildAgentServiceAccount(routeCR, commonLabels)
	if err = ctrl.SetControllerReference(&routeCR, desiredSAccount, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for ServiceAccount", "ServiceAccount.Name", desiredSAccount.Name)
		return ctrl.Result{}, err
	}
	if err = r.reconcileAgentServiceAccount(ctx, desiredSAccount); err != nil {
		return ctrl.Result{}, err
	}

	/*
		Create and create cluster roles and bindings for routebird-agent
	*/
	desiredCRole, desiredCRBinding := buildAgentClusterRole(routeCR, desiredSAccount, commonLabels)

	// ClusterRoles and ClusterRoleBindings cannot have a namespaced resource (like a CR) as their owner given that
	// they are cluster-scoped resources
	if err = r.reconcileAgentClusterRoles(ctx, desiredCRole, desiredCRBinding); err != nil {
		return ctrl.Result{}, err
	}

	/*
		Create, set up owner reference and create daemon set for routebird-agent
	*/
	desiredDSet := buildAgentDaemonSet(routeCR, desiredCMap, desiredSAccount, commonLabels)
	if err = ctrl.SetControllerReference(&routeCR, desiredDSet, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for DaemonSet", "DaemonSet.Name", desiredDSet.Name)
		return ctrl.Result{}, err
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
