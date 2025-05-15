package k8s

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	v1 "k8s.io/client-go/listers/core/v1"
	discoveryv1Lister "k8s.io/client-go/listers/discovery/v1"
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
	epsLister discoveryv1Lister.EndpointSliceLister

	nodeName string
	logger   logr.Logger
}

func NewControlLoop(
	informerFactory informers.SharedInformerFactory,
	nodeName string,
	logger logr.Logger,
) ControlLoop {

	svcLister := informerFactory.Core().V1().Services().Lister()
	epsLister := informerFactory.Discovery().V1().EndpointSlices().Lister()

	return &controlLoop{
		svcLister: svcLister,
		epsLister: epsLister,
		nodeName:  nodeName,
		logger:    logger,
	}
}

func (r *controlLoop) Resync(ctx context.Context) error {
	// TODO(): the cache should sync before running to simulate a real control loop, but given that this is a mock of
	// TODO(): a real control loop, right now it's OK

	services, err := r.svcLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}
	for _, svc := range services {
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			r.logger.Info("Skipping non LoadBalancer service", "service", svc.Name)
			continue
		}

		if svc.Spec.ExternalTrafficPolicy != corev1.ServiceExternalTrafficPolicyTypeCluster {
			r.logger.Info("Skipping service with externalTrafficPolicy different from Cluster", "service", svc.Name)
			continue
		}

		if len(svc.Spec.Selector) == 0 {
			r.logger.Info("Skipping service without selector", "service", svc.Name)
			continue
		}

		svcIPs := make([]string, 0)
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				svcIPs = append(svcIPs, ingress.IP)
			}
		}

		if len(svcIPs) == 0 {
			r.logger.Info("Skipping service without LoadBalancer IP", "service", svc.Name)
			continue
		}

		selector := labels.Set(map[string]string{
			discoveryv1.LabelServiceName: svc.Name,
		}).AsSelector()

		epsForService, errEPSLister := r.epsLister.EndpointSlices(svc.Namespace).List(selector)
		if errEPSLister != nil {
			return fmt.Errorf("failed to list endpoint slices for service %s: %v", svc.Name, errEPSLister)
		}

		for _, eps := range epsForService {
			// Iterate over the endpoints within each EndpointSlice
			for _, endpoint := range eps.Endpoints {
				if endpoint.NodeName != nil && *endpoint.NodeName == r.nodeName {
					r.logger.Info("Resync", "endpointSlice", eps.Name, "node", r.nodeName)
					// No need to check further endpoints in this EndpointSlice
					break
				}
			}
		}

		r.logger.Info("Resyncing", "service", svc.Name)
	}

	return nil
}
