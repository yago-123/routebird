package main

import (
	"github.com/go-logr/logr"
	"log/slog"
	"os"
	"time"
)

func main() {
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := logr.FromSlogHandler(slogLogger.Handler())

	for {
		logger.Info("Doing agent stuff...")
		time.Sleep(5 * time.Second)
	}
}
