package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/codeql"
	"github.com/hohn/mrvacommander/pkg/common"
	"github.com/hohn/mrvacommander/pkg/qldbstore"
	"github.com/hohn/mrvacommander/pkg/queue"
	"github.com/hohn/mrvacommander/utils"

	"github.com/google/uuid"
)

/*
type RunnerSingle struct {
	queue queue.Queue
}

func NewAgentSingle(numWorkers int, v *Visibles) *RunnerSingle {
	r := RunnerSingle{queue: v.Queue}

	for id := 1; id <= numWorkers; id++ {
		go r.worker(id)
	}
	return &r
}

func (r *RunnerSingle) worker(wid int) {
	var job common.AnalyzeJob

	for {
		job = <-r.queue.Jobs()
		result, err := RunAnalysisJob(job)
		if err != nil {
			slog.Error("Failed to run analysis job", slog.Any("error", err))
			continue
		}
		r.queue.Results() <- result
	}
}
*/

const (
	workerMemoryMB = 2048 // 2 GB
)

func StartAndMonitorWorkers(ctx context.Context,
	artifacts artifactstore.Store,
	databases qldbstore.Store,
	queue queue.Queue,
	desiredWorkerCount int,
	wg *sync.WaitGroup) {

	var workerCount int
	if desiredWorkerCount > 0 {
		workerCount = desiredWorkerCount
		slog.Info("Starting fixed number of workers", slog.Int("count", workerCount))
	} else {
		workerCount = 1
		slog.Info("Starting preset number of workers", slog.Int("count", workerCount))
	}

	stopChans := make([]chan struct{}, workerCount)

	for i := 0; i < workerCount; i++ {
		stopChan := make(chan struct{})
		stopChans[i] = stopChan
		wg.Add(1)
		go RunWorker(ctx, artifacts, databases, queue, stopChan, wg)
	}

	// Wait for context cancellation
	<-ctx.Done()

	for _, stopChan := range stopChans {
		close(stopChan)
	}
}

// RunAnalysisJob runs a CodeQL analysis job (AnalyzeJob) returning an AnalyzeResult
func RunAnalysisJob(
	job queue.AnalyzeJob, artifacts artifactstore.Store, dbs qldbstore.Store) (queue.AnalyzeResult, error) {
	var result = queue.AnalyzeResult{
		Spec:           job.Spec,
		ResultCount:    0,
		ResultLocation: artifactstore.ArtifactLocation{},
		Status:         common.StatusFailed,
	}

	// Create a temporary directory
	tempDir := filepath.Join(os.TempDir(), uuid.New().String())
	if err := os.MkdirAll(tempDir, 0600); err != nil {
		return result, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Download the query pack as a byte slice
	queryPackData, err := artifacts.GetQueryPack(job.QueryPackLocation)
	if err != nil {
		return result, fmt.Errorf("failed to download query pack: %w", err)
	}

	// Write the query pack data to the filesystem
	queryPackArchivePath := filepath.Join(tempDir, "query-pack.tar.gz")
	if err := os.WriteFile(queryPackArchivePath, queryPackData, 0600); err != nil {
		return result, fmt.Errorf("failed to write query pack archive to disk: %w", err)
	}

	// Make a directory and extract the query pack
	queryPackPath := filepath.Join(tempDir, "pack")
	if err := os.Mkdir(queryPackPath, 0600); err != nil {
		return result, fmt.Errorf("failed to create query pack directory: %w", err)
	}
	if err := utils.UntarGz(queryPackArchivePath, queryPackPath); err != nil {
		return result, fmt.Errorf("failed to extract query pack: %w", err)
	}

	databaseData, err := dbs.GetDatabase(job.Spec.NameWithOwner)
	if err != nil {
		slog.Error("Failed to get database",
			slog.String("owner", job.Spec.Owner),
			slog.String("repo", job.Spec.Repo),
			slog.Int("session_id", job.Spec.SessionID),
			slog.String("operation", "GetDatabase"),
			slog.Any("error", err),
		)
		return result, fmt.Errorf("failed to get database for %s/%s: %w",
			job.Spec.Owner, job.Spec.Repo, err)
	}

	// Write the CodeQL database data to the filesystem
	databasePath := filepath.Join(tempDir, "database.zip")
	if err := os.WriteFile(databasePath, databaseData, 0600); err != nil {
		return result, fmt.Errorf("failed to write CodeQL database to disk: %w", err)
	}

	// Perform the CodeQL analysis
	runResult, err := codeql.RunQuery(databasePath, job.QueryLanguage, queryPackPath, tempDir)
	if err != nil {
		return result, fmt.Errorf("failed to run analysis: %w", err)
	}

	// Generate a ZIP archive containing SARIF and BQRS files
	resultsArchive, err := codeql.GenerateResultsZipArchive(runResult)
	if err != nil {
		return result, fmt.Errorf("failed to generate results archive: %w", err)
	}

	// Upload the archive to storage
	slog.Debug("Results archive size", slog.Int("size", len(resultsArchive)))
	resultsLocation, err := artifacts.SaveResult(job.Spec, resultsArchive)
	if err != nil {
		return result, fmt.Errorf("failed to save results archive: %w", err)
	}

	result = queue.AnalyzeResult{
		Spec:                 job.Spec,
		ResultCount:          runResult.ResultCount,
		ResultLocation:       resultsLocation,
		Status:               common.StatusSucceeded,
		SourceLocationPrefix: runResult.SourceLocationPrefix,
		DatabaseSHA:          runResult.DatabaseSHA,
	}

	return result, nil
}

// RunWorker runs a worker that processes jobs from queue
func RunWorker(ctx context.Context,
	artifacts artifactstore.Store,
	databases qldbstore.Store,
	queue queue.Queue,
	stopChan chan struct{},
	wg *sync.WaitGroup) {
	const (
		WORKER_COUNT_STOP_MESSAGE   = "Worker stopping due to reduction in worker count"
		WORKER_CONTEXT_STOP_MESSAGE = "Worker stopping due to context cancellation"
	)

	defer wg.Done()
	slog.Info("Worker started")
	for {
		select {
		case <-stopChan:
			slog.Info(WORKER_COUNT_STOP_MESSAGE)
			return
		case <-ctx.Done():
			slog.Info(WORKER_CONTEXT_STOP_MESSAGE)
			return
		default:
			select {
			case job, ok := <-queue.Jobs():
				if !ok {
					return
				}
				slog.Info("Running analysis job", slog.Any("job", job))
				result, err := RunAnalysisJob(job, artifacts, databases)
				if err != nil {
					slog.Error("Failed to run analysis job", slog.Any("error", err))
					continue
				}
				slog.Info("Analysis job completed", slog.Any("result", result))
				queue.Results() <- result
			case <-stopChan:
				slog.Info(WORKER_COUNT_STOP_MESSAGE)
				return
			case <-ctx.Done():
				slog.Info(WORKER_CONTEXT_STOP_MESSAGE)
				return
			}
		}
	}
}
