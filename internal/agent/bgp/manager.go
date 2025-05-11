package bgp

import (
	"github.com/yago-123/routebird/api/v1alphav1"
	"k8s.io/client-go/kubernetes"
)

// todo: handle BGP peers and route announcements

type Manager interface {
	AnnounceRoute(route string)
	WithdrawRoute(route string)
}

type manager struct {
	peers  []v1alphav1.BGPPeer
	client kubernetes.Interface
}

func NewManager(peers []v1alphav1.BGPPeer, client kubernetes.Interface) Manager {
	return &manager{
		peers:  peers,
		client: client,
	}
}

func (m *manager) AnnounceRoute(route string) {

}

func (m *manager) WithdrawRoute(route string) {

}

// SCHEDULER
