package bgp

import (
	cfg "github.com/yago-123/routebird/internal/agent/config"
	"k8s.io/client-go/kubernetes"
)

// todo: handle BGP peers and route announcements

type Manager interface {
	AnnounceRoute(route string)
	WithdrawRoute(route string)
}

type manager struct {
	peers  []cfg.Peer
	client kubernetes.Interface
}

func NewManager(peers []cfg.Peer, client kubernetes.Interface) Manager {
	return &manager{
		peers:  peers,
		client: client,
	}
}

func (m *manager) AnnounceRoute(route string) {

}

func (m *manager) WithdrawRoute(route string) {

}
