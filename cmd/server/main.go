// Copyright © 2024 github
// Licensed under the Apache License, Version 2.0 (the "License").

package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"mrvacommander/config/mcc"

	"mrvacommander/pkg/agent"
	"mrvacommander/pkg/logger"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/server"
	"mrvacommander/pkg/storage"
)

func main() {
	// Define flags
	helpFlag := flag.Bool("help", false, "Display help message")
	logLevel := flag.String("loglevel", "info", "Set log level: debug, info, warn, error")
	mode := flag.String("mode", "standalone", "Set mode: standalone, container, cluster")

	// Custom usage function for the help flag
	flag.Usage = func() {
		log.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		log.Println("\nExamples:")
		log.Println("  go run main.go --loglevel=debug --mode=container")
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

	// Read configuration
	config := mcc.LoadConfig("mcconfig.toml")

	// Output configuration summary
	log.Printf("Help: %t\n", *helpFlag)
	log.Printf("Log Level: %s\n", *logLevel)
	log.Printf("Mode: %s\n", *mode)

	// Apply 'mode' flag
	switch *mode {
	case "standalone":
		// Assemble single-process version
		sq := queue.NewQueueSingle(2) // FIXME take value from configuration
		sc := server.NewCommanderSingle(nil, sq)
		sl := logger.NewLoggerSingle()
		ss := storage.NewStorageSingle(config.Storage.StartingID)
		sr := agent.NewRunnerSingle(2, sq) // FIXME take value from configuration

		qp, err := storage.NewQueryPackStore(config.Storage.StartingID)
		if err != nil {
			slog.Error("Unable to initialize query pack storage")
			os.Exit(1)
		}

		ql, err := storage.NewQLDBStore()
		if err != nil {
			slog.Error("Unable to initialize ql database storage")
			os.Exit(1)
		}

		sc.Setup(&server.CommanderVisibles{
			Logger:         sl,
			Queue:          sq,
			ServerStore:    ss,
			QueryPackStore: qp,
			QLDBStore:      ql,
		})

		sl.Setup(&logger.LoggerVisibles{})

		sq.Setup(&queue.QueueVisibles{
			Logger: sl,
		})

		ss.Setup(&storage.ServerStorageVisibles{})

		sr.Setup(&agent.RunnerVisibles{
			Logger:         sl,
			Queue:          sq,
			QueryPackStore: qp,
			QLDBStore:      ql,
		})

	case "container":
		// Assemble container version
		sq := queue.NewQueueSingle(2) // FIXME take value from configuration
		sc := server.NewCommanderSingle(nil, sq)
		sl := logger.NewLoggerSingle()

		ss, err := storage.NewStorageContainer(config.Storage.StartingID)
		if err != nil {
			slog.Error("Unable to initialize server storage")
			os.Exit(1)
		}

		qp, err := storage.NewQueryPackStore(config.Storage.StartingID)
		if err != nil {
			slog.Error("Unable to initialize query pack storage")
			os.Exit(1)
		}

		ql, err := storage.NewQLDBStore()
		if err != nil {
			slog.Error("Unable to initialize ql database storage")
			os.Exit(1)
		}

		sr := agent.NewRunnerSingle(2, sq) // FIXME take value from configuration

		sc.Setup(&server.CommanderVisibles{
			Logger:         sl,
			Queue:          sq,
			ServerStore:    ss,
			QueryPackStore: qp,
			QLDBStore:      ql,
		})

		sl.Setup(&logger.LoggerVisibles{})

		sq.Setup(&queue.QueueVisibles{
			Logger: sl,
		})

		ss.Setup(&storage.ServerStorageVisibles{})

		sr.Setup(&agent.RunnerVisibles{
			Logger:         sl,
			Queue:          sq,
			QueryPackStore: qp,
			QLDBStore:      ql,
		})

	case "cluster":
		// Assemble cccluster
	default:
		slog.Error("Invalid value for --mode. Allowed values are: standalone, container, cluster\n")
		os.Exit(1)
	}

}
