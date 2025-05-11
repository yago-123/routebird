package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
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
	client  kubernetes.Interface
	eventCh chan<- Event

	nodeName string
	logger   logr.Logger
}

func NewWatcher(client kubernetes.Interface, eventCh chan<- Event, nodeName string, logger logr.Logger) Watcher {
	return &watcher{
		client:   client,
		eventCh:  eventCh,
		nodeName: nodeName,
		logger:   logger,
	}
}

func (w *watcher) Watch(ctx context.Context) error {
	factory := informers.NewSharedInformerFactoryWithOptions(
		w.client,
		time.Minute,
		informers.WithNamespace(metav1.NamespaceAll), // Adjust if you want namespace scoping
	)

	// Service Informer
	svcInformer := factory.Core().V1().Services().Informer()
	_, _ = svcInformer.AddEventHandler(w.newHandler())

	// Endpoints Informer
	epInformer := factory.Core().V1().Endpoints().Informer()
	_, _ = epInformer.AddEventHandler(w.newHandler())

	// EndpointSlices Informer
	epsInformer := factory.Discovery().V1().EndpointSlices().Informer()
	_, _ = epsInformer.AddEventHandler(w.newHandler())

	factory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), svcInformer.HasSynced, epInformer.HasSynced, epsInformer.HasSynced) {
		return fmt.Errorf("failed to sync caches")
	}

	<-ctx.Done() // Wait until context is cancelled
	return nil
}

func (w *watcher) newHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			w.sendEvent(EventAdd, obj)
		},
		UpdateFunc: func(_, newObj any) {
			w.sendEvent(EventUpdate, newObj)
		},
		DeleteFunc: func(obj any) {
			w.sendEvent(EventDelete, obj)
		},
	}
}

func (w *watcher) sendEvent(eventType EventType, obj any) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		fmt.Printf("Failed to get key for object: %v\n", err)
		return
	}
	w.eventCh <- Event{
		Type: eventType,
		Obj:  obj,
		Key:  key,
	}
}
