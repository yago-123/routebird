package k8s

import (
	"context"
)

// todo: watches for Node, Pod, Service, or CR changes

type ResourceWatcher interface {
	Start(ctx context.Context)
}
