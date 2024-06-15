package main

import (
	"mrvacommander/pkg/agent"
	"mrvacommander/pkg/queue"
	"os/signal"
	"strconv"
	"syscall"

	"flag"
	"os"
	"runtime"
	"sync"

	"github.com/elastic/go-sysinfo"
	"golang.org/x/exp/slog"
)

func calculateWorkers() int {
	const workerMemoryMB = 2048 // 2 GB

	host, err := sysinfo.Host()
	if err != nil {
		slog.Error("failed to get host info", "error", err)
		os.Exit(1)
	}

	memInfo, err := host.Memory()
	if err != nil {
		slog.Error("failed to get memory info", "error", err)
		os.Exit(1)
	}

	// Get available memory in MB
	totalMemoryMB := memInfo.Available / (1024 * 1024)

	// Ensure we have at least one worker
	workers := int(totalMemoryMB / workerMemoryMB)
	if workers < 1 {
		workers = 1
	}

	// Limit the number of workers to the number of CPUs
	cpuCount := runtime.NumCPU()
	if workers > cpuCount {
		workers = max(cpuCount, 1)
	}

	return workers
}

func main() {
	slog.Info("Starting agent")

	workerCount := flag.Int("workers", 0, "number of workers")
	flag.Parse()

	requiredEnvVars := []string{
		"MRVA_RABBITMQ_HOST",
		"MRVA_RABBITMQ_PORT",
		"MRVA_RABBITMQ_USER",
		"MRVA_RABBITMQ_PASSWORD",
		"CODEQL_JAVA_HOME",
		"CODEQL_CLI_PATH",
	}

	for _, envVar := range requiredEnvVars {
		if _, ok := os.LookupEnv(envVar); !ok {
			slog.Error("Missing required environment variable %s", envVar)
			os.Exit(1)
		}
	}

	rmqHost := os.Getenv("MRVA_RABBITMQ_HOST")
	rmqPort := os.Getenv("MRVA_RABBITMQ_PORT")
	rmqUser := os.Getenv("MRVA_RABBITMQ_USER")
	rmqPass := os.Getenv("MRVA_RABBITMQ_PASSWORD")

	rmqPortAsInt, err := strconv.Atoi(rmqPort)

	if err != nil {
		slog.Error("Failed to parse RabbitMQ port", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Initializing RabbitMQ queue")

	rabbitMQQueue, err := queue.InitializeRabbitMQQueue(rmqHost, int16(rmqPortAsInt), rmqUser, rmqPass)
	if err != nil {
		slog.Error("failed to initialize RabbitMQ", slog.Any("error", err))
		os.Exit(1)
	}
	defer rabbitMQQueue.Close()

	if *workerCount == 0 {
		*workerCount = calculateWorkers()
	}

	slog.Info("Starting workers", slog.Int("count", *workerCount))
	var wg sync.WaitGroup
	for i := 0; i < *workerCount; i++ {
		wg.Add(1)
		go agent.RunWorker(rabbitMQQueue, &wg)
	}

	slog.Info("Agent startup complete")

	// Gracefully exit on SIGINT/SIGTERM (TODO: add job cleanup)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down agent")
		rabbitMQQueue.Close()
		os.Exit(0)
	}()

	select {}
}
