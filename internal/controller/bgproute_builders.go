package controller

import (
	"fmt"

	bgpv1alphav1 "github.com/yago-123/routebird/api/v1alphav1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RBACAPIGroup = "rbac.authorization.k8s.io"

	ClusterRoleKind    = "ClusterRole"
	ServiceAccountKind = "ServiceAccount"
)

func buildClusterRole(_ bgpv1alphav1.BGPRoute, roleName string) rbacv1.ClusterRole {
	return rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
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
}

func buildClusterRoleBinding(route bgpv1alphav1.BGPRoute, roleName, saName string) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName + "-binding",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: RBACAPIGroup,
			Kind:     ClusterRoleKind,
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      ServiceAccountKind,
				Name:      saName,
				Namespace: route.Namespace,
			},
		},
	}
}

func buildRoutebirdAgentServiceAccount(route bgpv1alphav1.BGPRoute, saName string) corev1.ServiceAccount {
	return corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: route.Namespace,
			// todo(): unify labels with DaemonSet
			Labels: map[string]string{
				"app":   "routebird-agent",
				"route": route.Name,
			},
		},
	}
}

func buildRoutebirdAgentDaemonSet(route bgpv1alphav1.BGPRoute, configMapName string, configMapHash string) appsv1.DaemonSet {
	// todo(): unify labels with ServiceAccount
	labels := map[string]string{"app": "routebird-agent", "route": route.Name}

	image := fmt.Sprintf("%s:%s", route.Spec.Agent.Image, route.Spec.Agent.Version)

	return appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "routebird-agent-" + route.Name,
			Namespace:   route.Namespace,
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
					ServiceAccountName: route.Spec.Agent.ServiceAccountName,
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
