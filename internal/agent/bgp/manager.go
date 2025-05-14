package bgp

import (
	"github.com/go-logr/logr"
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
	logger logr.Logger
}

func NewManager(peers []v1alphav1.BGPPeer, client kubernetes.Interface, logger logr.Logger) Manager {
	return &manager{
		peers:  peers,
		client: client,
		logger: logger,
	}
}

func (m *manager) AnnounceRoute(route string) {
	m.logger.Info("Announcing route", "route", route)
}

func (m *manager) WithdrawRoute(route string) {
	m.logger.Info("Withdrawing route", "route", route)
}
