package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/elastic/go-sysinfo"
	"golang.org/x/exp/slog"

	"mrvacommander/pkg/agent"
	"mrvacommander/pkg/queue"
)

const (
	workerMemoryMB     = 2048 // 2 GB
	monitorIntervalSec = 10   // Monitor every 10 seconds
)

func calculateWorkers() int {
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

func startAndMonitorWorkers(ctx context.Context, queue queue.Queue, desiredWorkerCount int, wg *sync.WaitGroup) {
	currentWorkerCount := 0
	stopChans := make([]chan struct{}, 0)

	if desiredWorkerCount != 0 {
		slog.Info("Starting workers", slog.Int("count", desiredWorkerCount))
		for i := 0; i < desiredWorkerCount; i++ {
			stopChan := make(chan struct{})
			stopChans = append(stopChans, stopChan)
			wg.Add(1)
			go agent.RunWorker(ctx, stopChan, queue, wg)
		}
		return
	}

	slog.Info("Worker count not specified, managing based on available memory and CPU")

	for {
		select {
		case <-ctx.Done():
			// signal all workers to stop
			for _, stopChan := range stopChans {
				close(stopChan)
			}
			return
		default:
			newWorkerCount := calculateWorkers()

			if newWorkerCount != currentWorkerCount {
				slog.Info(
					"Modifying worker count",
					slog.Int("current", currentWorkerCount),
					slog.Int("new", newWorkerCount))
			}

			if newWorkerCount > currentWorkerCount {
				for i := currentWorkerCount; i < newWorkerCount; i++ {
					stopChan := make(chan struct{})
					stopChans = append(stopChans, stopChan)
					wg.Add(1)
					go agent.RunWorker(ctx, stopChan, queue, wg)
				}
			} else if newWorkerCount < currentWorkerCount {
				for i := newWorkerCount; i < currentWorkerCount; i++ {
					close(stopChans[i])
				}
				stopChans = stopChans[:newWorkerCount]
			}
			currentWorkerCount = newWorkerCount

			time.Sleep(monitorIntervalSec * time.Second)
		}
	}
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
			slog.Error("Missing required environment variable", "key", envVar)
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

	rabbitMQQueue, err := queue.InitializeRabbitMQQueue(rmqHost, int16(rmqPortAsInt), rmqUser, rmqPass, false)
	if err != nil {
		slog.Error("failed to initialize RabbitMQ", slog.Any("error", err))
		os.Exit(1)
	}
	defer rabbitMQQueue.Close()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	go startAndMonitorWorkers(ctx, rabbitMQQueue, *workerCount, &wg)

	slog.Info("Agent started")

	// Gracefully exit on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	slog.Info("Shutting down agent")

	// TODO: fix this to gracefully terminate agent workers during jobs
	cancel()
	wg.Wait()

	slog.Info("Agent shutdown complete")
}
