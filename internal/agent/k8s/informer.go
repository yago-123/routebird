package k8s

import (
	"context"
	"fmt"

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
	epInformer  cache.SharedIndexInformer
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
	_, _ = svcInformer.AddEventHandler(newHandler(eventCh))

	// Endpoints Watcher
	epInformer := informerFactory.Core().V1().Endpoints().Informer()
	_, _ = epInformer.AddEventHandler(newHandler(eventCh))

	// EndpointSlices Watcher
	epsInformer := informerFactory.Discovery().V1().EndpointSlices().Informer()
	_, _ = epsInformer.AddEventHandler(newHandler(eventCh))

	return &watcher{
		informerFactory: informerFactory,
		svcInformer:     svcInformer,
		epInformer:      epInformer,
		epsInformer:     epsInformer,
		eventCh:         eventCh,
		nodeName:        nodeName,
		logger:          logger,
	}
}

func (w *watcher) Watch(ctx context.Context) error {
	// todo: recheck, this most likely is wrong
	w.informerFactory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), w.svcInformer.HasSynced, w.epInformer.HasSynced, w.epsInformer.HasSynced) {
		return fmt.Errorf("failed to sync caches")
	}

	<-ctx.Done() // Wait until context is cancelled
	return nil
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
