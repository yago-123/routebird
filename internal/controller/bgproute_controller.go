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
	"encoding/json"
	"fmt"
	"github.com/yago-123/routebird/internal/common"
	"reflect"

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

const ()

// BGPRouteReconciler reconciles a BGPRoute object
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
		err = r.Create(ctx, cfgMap)
		if err != nil {
			logger.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}

		logger.Info("Created ConfigMap", "ConfigMap.Name", cfgMap.Name)
	}

	// If contains error and is not NotFound, return error
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	// If already existed and it is not equal to the new one, update it
	if err == nil && !reflect.DeepEqual(existingCM.Data, cfgMap.Data) {
		existingCM.Data = cfgMap.Data
		err = r.Update(ctx, &existingCM)
		if err != nil {
			logger.Error(err, "Failed to update ConfigMap")
			return ctrl.Result{}, err
		}
		logger.Info("Updated ConfigMap", "ConfigMap.Name", cfgMap.Name)
	}

	// Define the desired DaemonSet
	newDSAgent := buildDaemonSet(route, cfgMap.Name)

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
		Named("bgproute").
		Complete(r)
}

func buildDaemonSet(route bgpv1alphav1.BGPRoute, configMapName string) appsv1.DaemonSet {
	labels := map[string]string{"app": "routebird-agent", "route": route.Name}

	image := fmt.Sprintf("yagodev123/routebird-agent:%s", route.Spec.Agent.Version)

	return appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "routebird-agent-" + route.Name,
			Namespace: route.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					// HostNetwork must be true in order to bind to the host's network
					HostNetwork: true,
					Containers: []corev1.Container{
						{
							Name:  "routebird-agent",
							Image: image,
							// todo(): make constant?
							Args: []string{"--config", "/etc/routebird/config.json"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/routebird",
									ReadOnly:  true,
								},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: route.Spec.BGPLocalPort, Name: "bgp", Protocol: corev1.ProtocolTCP},
							},
							ImagePullPolicy: route.Spec.Agent.ImagePullPolicy,
						},
					},
					// Mount the ConfigMap as a volume so that it can be accessed by the agent
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
					// Filter in which nodes the agent will run
					NodeSelector: route.Spec.NodeSelector,
					Tolerations:  route.Spec.Tolerations,
				},
			},
		},
	}
}
