// Copyright © 2024 github
// Licensed under the Apache License, Version 2.0 (the "License").

package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/advanced-security/mrvacommander/intefaces/mci"
	"github.com/advanced-security/mrvacommander/lib/commander/lcmem"
	"github.com/advanced-security/mrvacommander/lib/logger/llmem"
	"github.com/advanced-security/mrvacommander/lib/queue/lqmem"
	"github.com/advanced-security/mrvacommander/lib/runner/lrmem"
	"github.com/advanced-security/mrvacommander/lib/storage/lsmem"
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
		log.Println("  go run main.go --loglevel=Debug --mode=container")
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
	config := loadConfig("cconfig.toml")

	// Apply 'mode' flag
	switch *mode {
	case "standalone":
		// Assemble single-process version
		state := MCState{
			commander: lcmem.LCCommander,
			logger:    llmem.LCLogger,
			queue:     lqmem.LCQueue,
			storage:   lsmem.LCStorage,
			runner:    lrmem.LCRunner,
		}

	case "container":
		// Assemble cccontainer
	case "cluster":
		// Assemble cccluster
	default:
		slog.Error("Invalid value for --mode. Allowed values are: standalone, container, cluster\n")
		os.Exit(1)
	}

	// Output configuration summary
	log.Printf("Help: %t\n", *helpFlag)
	log.Printf("Log Level: %s\n", *logLevel)
	log.Printf("Mode: %s\n", *mode)

	// Run in the chosen mode

}

type MCCConf struct {
}
type MCLConf struct {
}
type MCQConf struct {
}
type MCSConf struct {
}
type MCRConf struct {
}

type MCState struct {
	commander mci.MCCommander
	logger    mci.MCLogger
	queue     mci.MCQueue
	storage   mci.MCStorage
	runner    mci.MCRunner
}

type MCConfig struct {
	commander MCCConf
	logger    MCLConf
	queue     MCQConf
	storage   MCSConf
	runner    MCRConf
}

func loadConfig(fname string) *MCConfig {
	if _, err := os.Stat(fname); err != nil {
		slog.Error("Configuration file %s not found", f)
		os.Exit(1)
	}

	var config MCConfig

	_, err := toml.DecodeFile(fname, &config)
	if err != nil {
		slog.Error("", err)
		os.Exit(1)
	}

	return &config
}
