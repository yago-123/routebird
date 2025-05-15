package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// todo: watches for Node, Pod, Service, or CR changes

type Watcher interface {
	// Watch subscribes to relevant Kubernetes API events (e.g., Services, Endpoints) and updates the internal desired
	// state accordingly.
	//
	// This method should run continuously until the provided context is canceled. Upon detecting changes, it should
	// notify the reconciliation process to promptly align BGP route advertisements with the updated state.
	Watch(ctx context.Context) error
}

// Service watcher must watch for
// - spec.type == LoadBalancer
// - service selector from CRD
// On add/update/delete
// Cluster-wide service watcher

// Endpoint/EndpointSlice watcher must watch for
// - only those specs that match the Service being served
// Npde-wide service watcher

type watcher struct {
	informerFactory informers.SharedInformerFactory

	svcInformer cache.SharedIndexInformer
	epsInformer cache.SharedIndexInformer

	// todo: really needed to be stored?
	eventCh chan<- Event

	nodeName string
	logger   logr.Logger
}

func NewWatcher(
	informerFactory informers.SharedInformerFactory,
	eventCh chan<- Event,
	nodeName string,
	logger logr.Logger,
) Watcher {

	// Service Watcher
	svcInformer := informerFactory.Core().V1().Services().Informer()

	// EndpointSlices Watcher
	epsInformer := informerFactory.Discovery().V1().EndpointSlices().Informer()

	return &watcher{
		informerFactory: informerFactory,
		svcInformer:     svcInformer,
		epsInformer:     epsInformer,
		eventCh:         eventCh,
		nodeName:        nodeName,
		logger:          logger,
	}
}

func (w *watcher) Watch(ctx context.Context) error {
	svcEventRegistration, err := w.svcInformer.AddEventHandler(newHandler(w.eventCh))
	if err != nil {
		return fmt.Errorf("failed to add service event handler: %w", err)
	}

	epsEventRegistration, err := w.epsInformer.AddEventHandler(newHandler(w.eventCh))
	if err != nil {
		return fmt.Errorf("failed to add endpoint slices event handler: %w", err)
	}

	<-ctx.Done() // Wait until context is cancelled

	err = w.svcInformer.RemoveEventHandler(svcEventRegistration)
	if err != nil {
		return fmt.Errorf("failed to remove service event handler: %w", err)
	}

	err = w.epsInformer.RemoveEventHandler(epsEventRegistration)
	if err != nil {
		return fmt.Errorf("failed to remove endpoint slices event handler: %w", err)
	}

	return nil
}

func newHandlerSvc(eventCh chan<- Event, logger logr.Logger) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if svc, ok := obj.(*corev1.Service); ok {
				// Only support LoadBalancer type services
				if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
					logger.Info("Service different from LoadBalancer", "service", svc.Name)
					return
				}

				if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {

				}

				if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyCluster {

				}

				// Only support services with selectors
				if len(svc.Spec.Selector) == 0 {
					logger.Info("Service has no selector", "service", svc.Name)
					return
				}

				sendEvent(EventAdd, obj, eventCh)
			}
		},
		UpdateFunc: func(_, newObj any) {
			sendEvent(EventUpdate, newObj, eventCh)
		},
		DeleteFunc: func(obj any) {
			sendEvent(EventDelete, obj, eventCh)
		},
	}
}

func newHandler(eventCh chan<- Event) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			sendEvent(EventAdd, obj, eventCh)
		},
		UpdateFunc: func(_, newObj any) {
			sendEvent(EventUpdate, newObj, eventCh)
		},
		DeleteFunc: func(obj any) {
			sendEvent(EventDelete, obj, eventCh)
		},
	}
}

func sendEvent(eventType EventType, obj any, eventCh chan<- Event) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		fmt.Printf("Failed to get key for object: %v\n", err)
		return
	}

	eventCh <- Event{
		Type: eventType,
		Obj:  obj,
		Key:  key,
	}
}
