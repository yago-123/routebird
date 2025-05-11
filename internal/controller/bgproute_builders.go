package controller

import (
	"encoding/json"
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bgpv1alphav1 "github.com/yago-123/routebird/api/v1alphav1"
	"github.com/yago-123/routebird/internal/common"
)

const (
	RBACAPIGroup = "rbac.authorization.k8s.io"

	ClusterRoleKind    = "ClusterRole"
	ServiceAccountKind = "ServiceAccount"
)

func buildAgentConfigMap(routeCR bgpv1alphav1.BGPRoute) (*corev1.ConfigMap, error) {
	cfg := common.Config{
		ServiceSelector: routeCR.Spec.ServiceSelector,
		LocalASN:        routeCR.Spec.LocalASN,
		BGPLocalPort:    routeCR.Spec.BGPLocalPort,
		Peers:           routeCR.Spec.Peers,
	}

	cfgJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal config to JSON: %w", err)
	}

	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", routeCR.Name),
			Namespace: routeCR.Namespace,
		},
		Data: map[string]string{
			"config.json": string(cfgJSON),
		},
	}

	return cfgMap, nil
}

func buildAgentServiceAccount(routeCR bgpv1alphav1.BGPRoute) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeCR.Spec.Agent.ServiceAccountName,
			Namespace: routeCR.Namespace,
			// todo(): unify labels with DaemonSet
			Labels: map[string]string{
				"app":   "routebird-agent",
				"route": routeCR.Name,
			},
		},
	}
}

func buildAgentClusterRole(routeCR bgpv1alphav1.BGPRoute, serviceAccount *corev1.ServiceAccount) (*rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding) {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: routeCR.Spec.Agent.ServiceAccountName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"endpoints", "services"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: routeCR.Spec.Agent.ServiceAccountName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: RBACAPIGroup,
			Kind:     ClusterRoleKind,
			Name:     routeCR.Spec.Agent.ServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      ServiceAccountKind,
				Name:      serviceAccount.Name,
				Namespace: routeCR.Namespace,
			},
		},
	}

	return clusterRole, clusterRoleBinding
}

func buildAgentDaemonSet(routeCR bgpv1alphav1.BGPRoute, configMap *corev1.ConfigMap, serviceAccount *corev1.ServiceAccount) *appsv1.DaemonSet {
	// todo(): unify labels with ServiceAccount
	labels := map[string]string{"app": "routebird-agent", "route": routeCR.Name}

	image := fmt.Sprintf("%s:%s", routeCR.Spec.Agent.Image, routeCR.Spec.Agent.Version)

	configMapHash := calculateCMapHash(configMap.Data)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("routebird-agent-%s", routeCR.Name),
			Namespace:   routeCR.Namespace,
			Labels:      labels,
			Annotations: map[string]string{"configMapHash": configMapHash},
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
					HostNetwork:        true,
					ServiceAccountName: serviceAccount.Name,
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
								{ContainerPort: routeCR.Spec.BGPLocalPort, Name: "bgp", Protocol: corev1.ProtocolTCP},
							},
							ImagePullPolicy: routeCR.Spec.Agent.ImagePullPolicy,
						},
					},
					// Mount the ConfigMap as a volume so that it can be accessed by the agent
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMap.Name,
									},
								},
							},
						},
					},
					// Filter in which nodes the agent will run
					NodeSelector: routeCR.Spec.NodeSelector,
					Tolerations:  routeCR.Spec.Tolerations,
				},
			},
		},
	}
}
