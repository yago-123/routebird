package k8s

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	v1 "k8s.io/client-go/listers/core/v1"
	discoveryv1 "k8s.io/client-go/listers/discovery/v1"
)

type ControlLoop interface {
	// Resync performs a full reconciliation to ensure that BGP route advertisements match the desired state derived
	// from the current cluster state.
	//
	// This method is typically called periodically as a safety mechanism to correct any missed updates or recover
	// from transient failures.
	//
	// It returns an error if the resynchronization fails and should be retried.
	Resync(ctx context.Context) error
}

// TODO(): this implementation is flawed, the current impl. is just to mock a control loop

type controlLoop struct {
	svcLister v1.ServiceLister
	epLister  v1.EndpointsLister
	epsLister discoveryv1.EndpointSliceLister

	nodeName string
	logger   logr.Logger
}

func NewControlLoop(
	informerFactory informers.SharedInformerFactory,
	nodeName string,
	logger logr.Logger,
) ControlLoop {

	svcLister := informerFactory.Core().V1().Services().Lister()
	epLister := informerFactory.Core().V1().Endpoints().Lister()
	epsLister := informerFactory.Discovery().V1().EndpointSlices().Lister()

	return &controlLoop{
		svcLister: svcLister,
		epLister:  epLister,
		epsLister: epsLister,
		nodeName:  nodeName,
		logger:    logger,
	}
}

func (r *controlLoop) Resync(ctx context.Context) error {
	// TODO(): the cache should sync before running to simulate a real control loop, but given that this is a mock of
	// TODO(): a real control loop, right now it's OK

	// todo(): filter eps that do not match selector
	endpoints, err := r.epLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list endpoints: %v", err)
	}
	for _, ep := range endpoints {
		for _, subset := range ep.Subsets {
			for _, address := range subset.Addresses {
				if address.NodeName != nil && *address.NodeName == r.nodeName {
					r.logger.Info("Resync", "endpoint", ep.Name, "node", r.nodeName)
				}
			}
		}
	}

	// todo(): filter eps that do not match selector
	endpointSlices, err := r.epsLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list endpoint slices: %v", err)
	}
	for _, eps := range endpointSlices {
		// Iterate over the endpoints within each EndpointSlice
		for _, endpoint := range eps.Endpoints {
			if endpoint.NodeName != nil && *endpoint.NodeName == r.nodeName {
				r.logger.Info("Resync", "endpointSlice", eps.Name, "node", r.nodeName)
				// No need to check further endpoints in this EndpointSlice
				break
			}
		}
	}

	return nil
}
