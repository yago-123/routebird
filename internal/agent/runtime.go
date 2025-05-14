package agent

import (
	"context"

	"github.com/yago-123/routebird/internal/agent/bgp"
	"github.com/yago-123/routebird/internal/agent/k8s"
	cfg "github.com/yago-123/routebird/internal/common"
	"k8s.io/client-go/kubernetes"
)

// todo: start watchers, BGP manager, handles event loop

type Runtime struct {
	bgpManager bgp.Manager
	watchers   []k8s.Watcher
}

func NewRuntime(cfg cfg.Config, client kubernetes.Interface) *Runtime {
	bgpManager := bgp.NewManager(cfg.Peers, client)

	watchers := []k8s.Watcher{
		// k8s.NewNodeWatcher(client, bgpManager),
		// k8s.NewCRDWatcher(client, bgpManager),
	}

	return &Runtime{bgpManager, watchers}
}

func (r *Runtime) Watch(ctx context.Context) error {
	// for _, w := range r.watchers {
	// 		go w.Watch(ctx)
	// }
	<-ctx.Done()
	return nil
}
