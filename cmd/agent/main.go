package main

import (
	"github.com/go-logr/logr"
	"log/slog"
	"os"
)

func main() {
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := logr.FromSlogHandler(slogLogger.Handler())

}
