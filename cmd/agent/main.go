package main

import (
	"io"
	"log"
	"mrvacommander/pkg/codeql"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/storage"
	"mrvacommander/utils"
	"net/http"
	"path/filepath"
	"runtime"

	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/exp/slog"

	"github.com/elastic/go-sysinfo"
)

func downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

func calculateWorkers() int {
	const workerMemoryGB = 2

	host, err := sysinfo.Host()
	if err != nil {
		log.Fatalf("failed to get host info: %v", err)
	}

	memInfo, err := host.Memory()
	if err != nil {
		log.Fatalf("failed to get memory info: %v", err)
	}

	// Convert total memory to GB
	totalMemoryGB := memInfo.Available / (1024 * 1024 * 1024)

	// Ensure we have at least one worker
	workers := int(totalMemoryGB / workerMemoryGB)
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

type RabbitMQQueue struct {
	jobs    chan common.AnalyzeJob
	results chan common.AnalyzeResult
	conn    *amqp.Connection
	channel *amqp.Channel
}

func InitializeQueue(jobsQueueName, resultsQueueName string) (*RabbitMQQueue, error) {
	rabbitMQHost := os.Getenv("MRVA_RABBITMQ_HOST")
	rabbitMQPort := os.Getenv("MRVA_RABBITMQ_PORT")
	rabbitMQUser := os.Getenv("MRVA_RABBITMQ_USER")
	rabbitMQPassword := os.Getenv("MRVA_RABBITMQ_PASSWORD")

	if rabbitMQHost == "" || rabbitMQPort == "" || rabbitMQUser == "" || rabbitMQPassword == "" {
		return nil, fmt.Errorf("RabbitMQ environment variables not set")
	}

	rabbitMQURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitMQUser, rabbitMQPassword, rabbitMQHost, rabbitMQPort)

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	_, err = ch.QueueDeclare(jobsQueueName, false, false, false, true, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare tasks queue: %w", err)
	}

	_, err = ch.QueueDeclare(resultsQueueName, false, false, false, true, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare results queue: %w", err)
	}

	err = ch.Qos(1, 0, false)

	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &RabbitMQQueue{
		conn:    conn,
		channel: ch,
		jobs:    make(chan common.AnalyzeJob),
		results: make(chan common.AnalyzeResult),
	}, nil
}

func (q *RabbitMQQueue) Jobs() chan common.AnalyzeJob {
	return q.jobs
}

func (q *RabbitMQQueue) Results() chan common.AnalyzeResult {
	return q.results
}

func (q *RabbitMQQueue) StartAnalyses(analysis_repos *map[common.NameWithOwner]storage.DBLocation, session_id int, session_language string) {
	slog.Info("Queueing codeql database analyze jobs")
}

func (q *RabbitMQQueue) Close() {
	q.channel.Close()
	q.conn.Close()
}

func (q *RabbitMQQueue) ConsumeJobs(queueName string) {
	msgs, err := q.channel.Consume(queueName, "", true, false, false, false, nil)
	if err != nil {
		slog.Error("failed to register a consumer", slog.Any("error", err))
	}

	for msg := range msgs {
		job := common.AnalyzeJob{}
		err := json.Unmarshal(msg.Body, &job)
		if err != nil {
			slog.Error("failed to unmarshal job", slog.Any("error", err))
			continue
		}
		q.jobs <- job
	}
	close(q.jobs)
}

func (q *RabbitMQQueue) PublishResults(queueName string) {
	for result := range q.results {
		q.publishResult(queueName, result)
	}
}

func (q *RabbitMQQueue) publishResult(queueName string, result interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultBytes, err := json.Marshal(result)
	if err != nil {
		slog.Error("failed to marshal result", slog.Any("error", err))
		return
	}

	slog.Info("Publishing result", slog.String("result", string(resultBytes)))
	err = q.channel.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        resultBytes,
		})
	if err != nil {
		slog.Error("failed to publish result", slog.Any("error", err))
	}
}

func RunAnalysisJob(job common.AnalyzeJob) (common.AnalyzeResult, error) {
	var result = common.AnalyzeResult{
		RequestId:        job.RequestId,
		ResultCount:      0,
		ResultArchiveURL: "",
		Status:           common.StatusError,
	}

	// Log job info
	slog.Info("Running analysis job", slog.Any("job", job))

	// Create a temporary directory
	tempDir := filepath.Join(os.TempDir(), uuid.New().String())
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return result, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract the query pack
	// TODO: download from the 'job' query pack URL
	utils.UntarGz("qp-54674.tgz", filepath.Join(tempDir, "qp-54674"))

	// Perform the CodeQL analysis
	runResult, err := codeql.RunQuery("google_flatbuffers_db.zip", "cpp", "qp-54674", tempDir)
	if err != nil {
		return result, fmt.Errorf("failed to run analysis: %w", err)
	}

	// Generate a ZIP archive containing SARIF and BQRS files
	resultsArchive, err := codeql.GenerateResultsZipArchive(runResult)
	if err != nil {
		return result, fmt.Errorf("failed to generate results archive: %w", err)
	}

	// TODO: Upload the archive to storage
	slog.Info("Results archive size", slog.Int("size", len(resultsArchive)))
	slog.Info("Analysis job successful.")

	result = common.AnalyzeResult{
		RequestId:        job.RequestId,
		ResultCount:      runResult.ResultCount,
		ResultArchiveURL: "REPLACE_THIS_WITH_STORED_RESULTS_ARCHIVE",
		Status:           common.StatusSuccess,
	}

	return result, nil
}

func RunWorker(queue queue.Queue, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range queue.Jobs() {
		result, err := RunAnalysisJob(job)
		if err != nil {
			slog.Error("failed to run analysis job", slog.Any("error", err))
			continue
		}
		queue.Results() <- result
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
		if os.Getenv(envVar) == "" {
			log.Fatalf("Fatal: Missing required environment variable %s", envVar)
		}
	}

	slog.Info("Initializing RabbitMQ connection")
	rabbitMQQueue, err := InitializeQueue("tasks", "results")
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
		go RunWorker(rabbitMQQueue, &wg)
	}

	slog.Info("Starting tasks consumer")
	go rabbitMQQueue.ConsumeJobs("tasks")

	slog.Info("Starting results publisher")
	go rabbitMQQueue.PublishResults("results")

	slog.Info("Agent startup complete")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down agent")
	close(rabbitMQQueue.results)
}
