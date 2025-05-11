package main

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/yago-123/routebird/internal/agent/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"log/slog"
	"os"
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

	w := k8s.NewWatcher(clientset, eventCh, nodeName, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if errWatch := w.Watch(ctx); errWatch != nil {
			logger.Error(errWatch, "Failed to watch resources")
		}
	}()

	// Event processing loop
	for event := range eventCh {
		logger.Info("Received event", "type", event.Type, "key", event.Key)
		// todo: trigger reconciliation or BGP updates here
	}
}
