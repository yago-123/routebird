package k8s

import (
	"context"
)

type Resyncer interface {
	// Resync performs a full reconciliation to ensure that BGP route advertisements match the desired state derived
	// from the current cluster state.
	//
	// This method is typically called periodically as a safety mechanism to correct any missed updates or recover
	// from transient failures.
	//
	// It returns an error if the resynchronization fails and should be retried.
	Resync(ctx context.Context) error
}

type resyncer struct {
}

func NewReconciler() Resyncer {
	return &resyncer{}
}

func (r *resyncer) Resync(_ context.Context) error {
	return nil
}
