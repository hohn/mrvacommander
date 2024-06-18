// Copyright © 2024 github
// Licensed under the Apache License, Version 2.0 (the "License").

package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"strconv"

	"mrvacommander/config/mcc"

	"mrvacommander/pkg/agent"
	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/qldbstore"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/server"
	"mrvacommander/pkg/state"
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
		sq := queue.NewQueueSingle(2)
		ss := state.NewLocalState(config.Storage.StartingID)
		as := artifactstore.NewInMemoryArtifactStore()
		ql := qldbstore.NewLocalFilesystemCodeQLDatabaseStore("")

		server.NewCommanderSingle(&server.Visibles{
			Queue:         sq,
			State:         ss,
			Artifacts:     as,
			CodeQLDBStore: ql,
		})

		// FIXME take value from configuration
		agent.NewAgentSingle(2, &agent.Visibles{
			Queue:         sq,
			Artifacts:     as,
			CodeQLDBStore: ql,
		})

	case "container":
		// TODO: take value from configuration

		rmqHost := os.Getenv("MRVA_RABBITMQ_HOST")
		rmqPort := os.Getenv("MRVA_RABBITMQ_PORT")
		rmqUser := os.Getenv("MRVA_RABBITMQ_USER")
		rmqPass := os.Getenv("MRVA_RABBITMQ_PASSWORD")

		rmqPortAsInt, err := strconv.ParseInt(rmqPort, 10, 16)
		if err != nil {
			slog.Error("Failed to parse RabbitMQ port", slog.Any("error", err))
			os.Exit(1)
		}

		sq, err := queue.NewRabbitMQQueue(rmqHost, int16(rmqPortAsInt), rmqUser, rmqPass, false)
		if err != nil {
			slog.Error("Unable to initialize RabbitMQ queue")
			os.Exit(1)
		}

		ss := state.NewLocalState(config.Storage.StartingID)

		as, err := artifactstore.NewMinIOArtifactStore("", "", "") // TODO: add arguments
		if err != nil {
			slog.Error("Unable to initialize artifact store")
			os.Exit(1)
		}

		ql, err := qldbstore.NewMinIOCodeQLDatabaseStore("", "", "", "")
		if err != nil {
			slog.Error("Unable to initialize ql database storage")
			os.Exit(1)
		}

		server.NewCommanderSingle(&server.Visibles{
			Queue:         sq,
			State:         ss,
			Artifacts:     as,
			CodeQLDBStore: ql,
		})

	case "cluster":
		// Assemble cluster version
	default:
		slog.Error("Invalid value for --mode. Allowed values are: standalone, container, cluster\n")
		os.Exit(1)
	}

}
