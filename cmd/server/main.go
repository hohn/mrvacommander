// Copyright © 2024 github
// Licensed under the Apache License, Version 2.0 (the "License").

package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/hohn/mrvacommander/config/mcc"

	"github.com/hohn/mrvacommander/pkg/agent"
	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/deploy"
	"github.com/hohn/mrvacommander/pkg/qldbstore"
	"github.com/hohn/mrvacommander/pkg/queue"
	"github.com/hohn/mrvacommander/pkg/server"
	"github.com/hohn/mrvacommander/pkg/state"
)

func main() {
	// Define flags
	helpFlag := flag.Bool("help", false, "Display help message")
	logLevel := flag.String("loglevel", "info", "Set log level: debug, info, warn, error")
	mode := flag.String("mode", "standalone", "Set mode: standalone, container, cluster")
	dbPathRoot := flag.String("dbpath", "", "Set the root path for the database store if using standalone mode.")

	// Custom usage function for the help flag
	flag.Usage = func() {
		log.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		log.Println("\nExamples:")
		log.Println("go run main.go --loglevel=debug --mode=container --dbpath=/path/to/db_dir")
	}

	// Parse the flags
	flag.Parse()

	// Handle the help flag
	if *helpFlag {
		flag.Usage()
		return
	}

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

	// Process database root if standalone and not provided
	if *mode == "standalone" && *dbPathRoot == "" {
		slog.Warn("No database root path provided.")
		// Current directory of the Executable has a codeql directory. There.
		// Resolve the absolute directory based on os.Executable()
		execPath, err := os.Executable()
		if err != nil {
			slog.Error("Failed to get executable path", slog.Any("error", err))
			os.Exit(1)
		}
		*dbPathRoot = filepath.Dir(execPath) + "/codeql/dbs/"
		slog.Info("Using default database root path", "dbPathRoot", *dbPathRoot)
	}

	// Read configuration
	config := mcc.LoadConfig("mcconfig.toml")

	// Output configuration summary
	log.Printf("Help: %t\n", *helpFlag)
	log.Printf("Log Level: %s\n", *logLevel)
	log.Printf("Mode: %s\n", *mode)

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Apply 'mode' flag
	switch *mode {
	case "standalone":
		// Assemble single-process version
		sq := queue.NewQueueSingle(2)
		ss := state.NewLocalState(config.Storage.StartingID)
		as := artifactstore.NewInMemoryArtifactStore()
		ql := qldbstore.NewLocalFilesystemCodeQLDatabaseStore(*dbPathRoot)

		server.NewCommanderSingle(&server.Visibles{
			Queue:         sq,
			State:         ss,
			Artifacts:     as,
			CodeQLDBStore: ql,
		})

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())

		go agent.StartAndMonitorWorkers(ctx, as, ql, sq, 2, &wg)

		slog.Info("Started server and standalone agent")
		<-sigChan
		slog.Info("Shutting down...")
		cancel()
		wg.Wait()
		slog.Info("Agent shutdown complete")

	case "container":
		isAgent := false

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

		server.NewCommanderSingle(&server.Visibles{
			Queue:         rabbitMQQueue,
			State:         state.NewLocalState(config.Storage.StartingID),
			Artifacts:     artifacts,
			CodeQLDBStore: databases,
		})

		slog.Info("Started server in container mode.")
		<-sigChan
	default:
		slog.Error("Invalid value for --mode. Allowed values are: standalone, container, cluster")
		os.Exit(1)
	}

	slog.Info("Server shutdown complete")
}
