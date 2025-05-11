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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/yago-123/routebird/internal/common"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// Retrieve the BGPRoute instance
	if err := r.Get(ctx, req.NamespacedName, &route); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Generate the configuration for the BGP agent
	// todo(): move to a separate function
	cfg := common.Config{
		ServiceSelector: route.Spec.ServiceSelector,
		LocalASN:        route.Spec.LocalASN,
		BGPLocalPort:    route.Spec.BGPLocalPort,
		Peers:           route.Spec.Peers,
	}
	cfgJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		logger.Error(err, "Failed to marshal config to JSON")
		return ctrl.Result{}, err
	}

	// Create or update the ConfigMap with the config
	// todo(): make to a separate function
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      route.Name + "-config",
			Namespace: route.Namespace,
		},
		Data: map[string]string{
			"config.json": string(cfgJSON),
		},
	}
	if err = ctrl.SetControllerReference(&route, cfgMap, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for ConfigMap")
		return ctrl.Result{}, err
	}

	var existingCM corev1.ConfigMap
	// Handle ConfigMap creation or update
	err = r.Get(ctx, client.ObjectKeyFromObject(cfgMap), &existingCM)
	// If not found, create it
	if apierrors.IsNotFound(err) {
		errCreate := r.Create(ctx, cfgMap)
		if err != nil {
			logger.Error(errCreate, "Failed to create ConfigMap")
			return ctrl.Result{}, errCreate
		}

		logger.Info("Created ConfigMap", "ConfigMap.Name", cfgMap.Name)
	} else if err != nil && !apierrors.IsNotFound(err) {
		// If contains error and is not NotFound, return error
		logger.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	// If already existed and it is not equal to the new one, update it
	if err == nil && !reflect.DeepEqual(existingCM.Data, cfgMap.Data) {
		existingCM.Data = cfgMap.Data
		errUpdate := r.Update(ctx, &existingCM)
		if errUpdate != nil {
			logger.Error(errUpdate, "Failed to update ConfigMap")
			return ctrl.Result{}, errUpdate
		}
		logger.Info("Updated ConfigMap", "ConfigMap.Name", cfgMap.Name)
	}

	// Build the ServiceAccount object for the DaemonSet
	sAccountName := route.Spec.Agent.ServiceAccountName
	newSAAgent := buildRoutebirdAgentServiceAccount(route, sAccountName)
	if err = ctrl.SetControllerReference(&route, &newSAAgent, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for ServiceAccount", "ServiceAccount.Name", newSAAgent.Name)
		return ctrl.Result{}, err
	}

	// Create or update the ServiceAccount
	var sa corev1.ServiceAccount
	err = r.Get(ctx, client.ObjectKey{Name: sAccountName, Namespace: route.Namespace}, &sa)
	if apierrors.IsNotFound(err) {
		if err = r.Create(ctx, &newSAAgent); err != nil {
			logger.Error(err, "Failed to create ServiceAccount", "ServiceAccount.Name", sAccountName)
			return ctrl.Result{}, err
		}
		logger.Info("Created ServiceAccount", "ServiceAccount.Name", sAccountName)
	} else if err != nil {
		logger.Error(err, "Failed to get ServiceAccount", "ServiceAccount.Name", sAccountName)
		return ctrl.Result{}, err
	}

	// todo(): create RBAC roles and bindings for the ServiceAccount
	_ = buildClusterRole(route, "routebird-agent")
	_ = buildClusterRoleBinding(route, "routebird-agent", sAccountName)

	// Calculate new config hash after ConfigMap update
	configMapHash := calculateConfigMapHash(cfgMap.Data)

	// Define the desired DaemonSet
	newDSAgent := buildRoutebirdAgentDaemonSet(route, cfgMap.Name, configMapHash)

	// Set owner reference to the DaemonSet
	if err = ctrl.SetControllerReference(&route, &newDSAgent, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference", "DaemonSet.Name", newDSAgent.Name)
		return ctrl.Result{}, err
	}

	// Check if the DaemonSet already exists
	var existingDSAgent appsv1.DaemonSet
	err = r.Get(ctx, client.ObjectKey{Name: newDSAgent.Name, Namespace: newDSAgent.Namespace}, &existingDSAgent)
	// If not found, create it
	if err != nil && apierrors.IsNotFound(err) {
		if errCreate := r.Create(ctx, &newDSAgent); errCreate != nil {
			logger.Error(errCreate, "Failed to create DaemonSet", "DaemonSet.Name", newDSAgent.Name)
			return ctrl.Result{}, errCreate
		}

		logger.Info("Created DaemonSet", "DaemonSet.name", newDSAgent.Name)
	}

	// If contains error and is not NotFound, return error
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to get DaemonSet", "DaemonSet.Name", newDSAgent.Name)
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

// calculateConfigMapHash generates a deterministic hash based on the ConfigMap's data content.
func calculateConfigMapHash(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	// Ensure consistent ordering
	sort.Strings(keys)

	hasher := sha256.New()
	for _, k := range keys {
		hasher.Write([]byte(k))
		hasher.Write([]byte(data[k]))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
