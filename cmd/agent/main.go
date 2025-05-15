package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"k8s.io/client-go/tools/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/go-logr/logr"
	"github.com/yago-123/routebird/internal/agent/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// todo(): set from config or env
	InformerResyncInterval = 1 * time.Minute
)

func main() {
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := logr.FromSlogHandler(slogLogger.Handler())

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	// nodeName := os.Getenv("NODE_NAME")
	nodeName := "minikube"
	eventCh := make(chan k8s.Event, 100)

	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		InformerResyncInterval,
		informers.WithNamespace(metav1.NamespaceAll),
	)

	svcInformer := informerFactory.Core().V1().Services().Informer()
	epsInformer := informerFactory.Discovery().V1().EndpointSlices().Informer()

	epsInformer.AddIndexers(cache.Indexers{
		"by-service": func(obj interface{}) ([]string, error) {
			// custom indexing logic

			return []string{}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	informerFactory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), svcInformer.HasSynced, epsInformer.HasSynced) {
		logger.Error(fmt.Errorf("Failed to sync caches"), "svcInformer", svcInformer, "epsInformer", epsInformer)
		return
	}

	watcher := k8s.NewWatcher(informerFactory, eventCh, nodeName, logger)
	controlLoop := k8s.NewControlLoop(informerFactory, nodeName, logger)

	go func() {
		if errWatch := watcher.Watch(ctx); errWatch != nil {
			logger.Error(errWatch, "Failed to watch resources")
		}
	}()

	go func() {
		for {
			controlLoop.Resync(context.Background())
			time.Sleep(10 * time.Second)
		}
	}()

	// Event processing loop
	for evt := range eventCh {
		// Print the event and the concrete type of Obj
		fmt.Printf("Got event %s for %T (key=%q)\n", evt.Type, evt.Obj, evt.Key)

		// If you need to drill in on specific fields:
		switch o := evt.Obj.(type) {
		case *corev1.Service:
			fmt.Printf("  Service: %s/%s, Type=%s, Selector=%v\n",
				o.Namespace, o.Name, o.Spec.Type, o.Spec.Selector)
		case *corev1.Endpoints:
			fmt.Printf("  Endpoints: %s/%s, Addresses=%v\n",
				o.Namespace, o.Name, o.Subsets)
		case *discoveryv1.EndpointSlice:
			// Collect all addresses across all Endpoint entries
			var addrs []string
			for _, ep := range o.Endpoints {
				addrs = append(addrs, ep.Addresses...)
			}
			fmt.Printf("  EndpointSlice: %s/%s, Addresses=%v\n",
				o.Namespace, o.Name, addrs)
		default:
			fmt.Printf("  Unknown object type: %T\n", o)
		}
	}
}
