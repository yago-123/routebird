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

package v1alphav1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BGPRouteSpec defines the desired state of BGPRoute.
type BGPRouteSpec struct {
	// ServiceSelector match services by label
	// +kubebuilder:default:={"matchLabels":{"__never_match__":"true"}}
	ServiceSelector metav1.LabelSelector `json:"serviceSelector"`

	// LocalASN of the node where the route is advertised
	// +kubebuilder:validation:Minimum=1
	LocalASN uint32 `json:"localASN"`

	// BGPLocalPort is the port used by the BGP agent to listen for incoming BGP connections
	// +kubebuilder:validation:Minimum=1
	BGPLocalPort int32 `json:"bgpLocalPort"`

	// Peers to which the route should be advertised
	// todo: think on whether might make sense to have 0 peers, since this is a P2P protocol
	// +kubebuilder:validation:MinItems=1
	Peers []BGPPeer `json:"bgpPeers,omitempty"`

	// Agent details for the route advertisement DaemonSet specification
	Agent Agent `json:"agent,omitempty"`

	// Filtering capabilities for the route advertisement
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
}

type BGPPeer struct {
	// todo: add options for DNS resolution
	// Address of the remote peer receiving BGP updates
	// +kubebuilder:validation:Pattern=`^([0-9a-fA-F:.]+)$`
	Address string `json:"address"`
	// ASN of the remote peer receiving BGP updates
	ASN uint32 `json:"asn"`
}

type Agent struct {
	// Version of the BGP agent that will announce routes
	// +kubebuilder:default="latest"
	Version string `json:"version"`

	// +kubebuilder:default="IfNotPresent"
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`
}

// BGPRouteStatus defines the observed state of BGPRoute.
type BGPRouteStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BGPRoute is the Schema for the bgproutes API.
type BGPRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BGPRouteSpec   `json:"spec,omitempty"`
	Status BGPRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BGPRouteList contains a list of BGPRoute.
type BGPRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BGPRoute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BGPRoute{}, &BGPRouteList{})
}
