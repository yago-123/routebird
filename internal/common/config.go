package common

import (
	"github.com/yago-123/routebird/api/v1alphav1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigMapPath     = "/routebird/config"
	ConfigMapFilename = "config.json"
)

// todo(): decide how to add versioning to this config struct
type Config struct {
	ServiceSelector metav1.LabelSelector
	LocalASN        uint32
	BGPLocalPort    int32
	Peers           []v1alphav1.BGPPeer
}
