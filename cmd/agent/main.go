package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hohn/mrvacommander/pkg/agent"
	"github.com/hohn/mrvacommander/pkg/deploy"
)

func main() {
	slog.Info("Starting agent")
	workerCount := flag.Int("workers", 0, "number of workers")
	logLevel := flag.String("loglevel", "info", "Set log level: debug, info, warn, error")
	flag.Parse()

	// Apply 'loglevel' flag
	switch *logLevel {
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		log.Printf("Invalid logging verbosity level: %s", *logLevel)
		os.Exit(1)
	}

	isAgent := true

	rabbitMQQueue, err := deploy.InitRabbitMQ(isAgent)
	if err != nil {
		slog.Error("Failed to initialize RabbitMQ", slog.Any("error", err))
		os.Exit(1)
	}
	defer rabbitMQQueue.Close()

	artifacts, err := deploy.InitMinIOArtifactStore()
	if err != nil {
		slog.Error("Failed to initialize artifact store", slog.Any("error", err))
		os.Exit(1)
	}

	databases, err := deploy.InitMinIOCodeQLDatabaseStore()
	if err != nil {
		slog.Error("Failed to initialize database store", slog.Any("error", err))
		os.Exit(1)
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	go agent.StartAndMonitorWorkers(ctx, artifacts, databases, rabbitMQQueue, *workerCount, &wg)
	slog.Info("Agent started")

	// Gracefully exit on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down agent")
	cancel()
	wg.Wait()
	slog.Info("Agent shutdown complete")
}
