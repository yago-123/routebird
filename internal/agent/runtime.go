package agent

import (
	"context"
	"github.com/yago-123/routebird/internal/agent/bgp"
	cfg "github.com/yago-123/routebird/internal/agent/config"
	"github.com/yago-123/routebird/internal/agent/k8s"
	"k8s.io/client-go/kubernetes"
)

// todo: start watchers, BGP manager, handles event loop

type Runtime struct {
	bgpManager bgp.Manager
	watchers   []k8s.ResourceWatcher
}

func NewRuntime(cfg cfg.Config, client kubernetes.Interface) *Runtime {
	bgpManager := bgp.NewManager(cfg.BGPPeers, client)

	watchers := []k8s.ResourceWatcher{
		// k8s.NewNodeWatcher(client, bgpManager),
		// k8s.NewCRDWatcher(client, bgpManager),
	}

	return &Runtime{bgpManager, watchers}
}

func (r *Runtime) Start(ctx context.Context) error {
	for _, w := range r.watchers {
		go w.Start(ctx)
	}
	<-ctx.Done()
	return nil
}
