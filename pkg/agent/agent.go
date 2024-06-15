package agent

import (
	"fmt"
	"log/slog"
	"mrvacommander/pkg/codeql"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/logger"
	"mrvacommander/pkg/qpstore"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/storage"
	"mrvacommander/utils"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type RunnerSingle struct {
	queue queue.Queue
}

func NewRunnerSingle(numWorkers int, queue queue.Queue) *RunnerSingle {
	r := RunnerSingle{queue: queue}

	for id := 1; id <= numWorkers; id++ {
		go r.worker(id)
	}
	return &r
}

type Visibles struct {
	Logger logger.Logger
	Queue  queue.Queue
	// TODO extra package for query pack storage
	QueryPackStore qpstore.Storage
	// TODO extra package for ql db storage
	QLDBStore storage.Storage
}

func (c *RunnerSingle) Setup(st *Visibles) {
	// TODO: implement
}

func (r *RunnerSingle) worker(wid int) {
	// TODO: reimplement this later
	/*
		var job common.AnalyzeJob

		for {
			job = <-r.queue.Jobs()

			slog.Debug("Picking up job", "job", job, "worker", wid)

			slog.Debug("Analysis: running", "job", job)
			storage.SetStatus(job.QueryPackId, job.NWO, common.StatusQueued)

			resultFile, err := RunAnalysis(job)
			if err != nil {
				continue
			}

			slog.Debug("Analysis run finished", "job", job)

			// TODO: FIX THIS
			res := common.AnalyzeResult{
				RunAnalysisSARIF: resultFile,
				RunAnalysisBQRS:  "", // FIXME ?
			}
			r.queue.Results() <- res
			storage.SetStatus(job.QueryPackId, job.NWO, common.StatusSuccess)
			storage.SetResult(job.QueryPackId, job.NWO, res)

		}
	*/
}

// RunAnalysisJob runs a CodeQL analysis job (AnalyzeJob) returning an AnalyzeResult
func RunAnalysisJob(job common.AnalyzeJob) (common.AnalyzeResult, error) {
	var result = common.AnalyzeResult{
		RequestId:        job.RequestId,
		ResultCount:      0,
		ResultArchiveURL: "",
		Status:           common.StatusError,
	}

	// Create a temporary directory
	tempDir := filepath.Join(os.TempDir(), uuid.New().String())
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return result, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract the query pack
	// TODO: download from the 'job' query pack URL
	// utils.downloadFile
	queryPackPath := filepath.Join(tempDir, "qp-54674")
	utils.UntarGz("qp-54674.tgz", queryPackPath)

	// Perform the CodeQL analysis
	runResult, err := codeql.RunQuery("google_flatbuffers_db.zip", "cpp", queryPackPath, tempDir)
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
		ResultArchiveURL: "REPLACE_THIS_WITH_STORED_RESULTS_ARCHIVE", // TODO
		Status:           common.StatusSuccess,
	}

	return result, nil
}

// RunWorker runs a worker that processes jobs from queue
func RunWorker(queue queue.Queue, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range queue.Jobs() {
		slog.Info("Running analysis job", slog.Any("job", job))
		result, err := RunAnalysisJob(job)
		if err != nil {
			slog.Error("Failed to run analysis job", slog.Any("error", err))
			continue
		}
		slog.Info("Analysis job completed", slog.Any("result", result))
		queue.Results() <- result
	}
}
