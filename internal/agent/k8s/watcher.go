package k8s

import (
	"context"
)

// todo: watches for Node, Pod, Service, or CR changes

type ResourceWatcher interface {
	Start(ctx context.Context)
}

// Service watcher must watch for
// - spec.type == LoadBalancer
// - service selector from CRD
// On add/update/delete
// Cluster-wide service watcher

// Endpoint/EndpointSlice watcher must watch for
// - only those specs that match the Service being served
// Npde-wide service watcher
