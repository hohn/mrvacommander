package agent

import (
	"context"
	"fmt"
	"log/slog"
	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/codeql"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/queue"
	"mrvacommander/utils"

	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

type RunnerSingle struct {
	v *Visibles
}

func NewAgentSingle(numWorkers int, v *Visibles) *RunnerSingle {
	slog.Debug("Agent started")

	r := RunnerSingle{v: v}

	for id := 1; id <= numWorkers; id++ {
		go r.worker(id)
	}
	return &r
}

func (r *RunnerSingle) worker(wid int) {
	slog.Debug("Worker started", "worker", wid)
	slog.Debug("Worker queue", "address",
		reflect.ValueOf(r.v.Queue.Jobs()).Pointer())

	var job common.AnalyzeJob

	for {
		job = <-r.v.Queue.Jobs()

		slog.Debug("Picked up job", "job", job, "worker", wid)

		slog.Debug("Analysis: running", "job", job)

		result, err := r.RunAnalysisJob(job)
		if err != nil {
			continue
		}

		slog.Debug("Analysis run finished", "job", job)

		res := common.AnalyzeResult{
			RequestId:   job.RequestId,
			ResultCount: result.ResultCount,
			// TODO get rid of string->string map
			ResultLocation: result.ResultLocation,
			Status:         common.StatusSuccess,
			NWO:            job.NWO,
		}
		r.v.Queue.Results() <- res
		slog.Debug("	XX: result queue push:", "res", res)
		// XX: StatusResponse needs to pick up this ^^^^ info.
		// XX: so astat := c.v.State.GetStatus(js.JobID, js.NameWithOwner).ToExternalString() works
	}
}

func (r *RunnerSingle) RunAnalysisJob(job common.AnalyzeJob) (common.AnalyzeResult, error) {
	var result = common.AnalyzeResult{
		RequestId:      job.RequestId,
		ResultCount:    0,
		ResultLocation: artifactstore.ArtifactLocation{},
		Status:         common.StatusError,
		NWO:            job.NWO,
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
	// XX: check
	qpKey := r.v.Artifacts.QPKeyFromID(job.QueryPackId)
	qpLocation := job.QueryPackLocation.PathFor(qpKey)
	qpExtractTo := os.Getenv("MRVA_QP_ROOT")
	utils.UntarGz(qpLocation, qpExtractTo)

	// Perform the CodeQL analysis
	// TODO XX: real path
	resultFile, err := r.RunAnalysis(job)
	if err != nil {
		slog.Error("Analysis job failed", "job", job.RequestId)
		return common.AnalyzeResult{}, err
	}

	// XX: check
	// Generate a ZIP archive containing SARIF and BQRS files
	sarif, err := os.ReadFile(resultFile)
	if err != nil {
		slog.Error("Failed to read SARIF file", "err", err)
	}
	resultCount := codeql.GetSarifResultCount(sarif)

	runResult := codeql.RunQueryResult{
		ResultCount:          resultCount,
		DatabaseSHA:          "TODO dbsha",
		SourceLocationPrefix: "TODO slp",
		BqrsFilePaths:        codeql.BqrsFilePaths{}, // TODO empty ok?
		SarifFilePath:        resultFile,
	}
	resultsArchive, err := codeql.GenerateResultsZipArchive(&runResult)
	if err != nil {
		return result, fmt.Errorf("failed to generate results archive: %w", err)
	}

	// XX: Save the archive in storage
	slog.Debug("Results archive info:", slog.Int("size", len(resultsArchive)))

	// XX: check
	serverRoot := os.Getenv("MRVA_SERVER_ROOT")
	queryOutDir := filepath.Join(serverRoot,
		"var/codeql/sarif/localrun", job.NWO.Owner, job.NWO.Repo)
	queryOutFName := fmt.Sprintf("results-%d.sarif", job.RequestId)
	queryOutFPath := filepath.Join(queryOutDir, queryOutFName)

	s := artifactstore.NewArtifactLocation()
	s.Add(queryOutFName, queryOutFPath)

	result = common.AnalyzeResult{
		RequestId:      job.RequestId,
		ResultCount:    runResult.ResultCount,
		ResultLocation: *s,
		Status:         common.StatusSuccess,
		NWO:            job.NWO,
	}

	return result, nil
}

// TODO /Maybe/ move to codeql package
func (r *RunnerSingle) RunAnalysis(job common.AnalyzeJob) (string, error) {
	// TODO Add multi-language tests including queryLanguage
	// queryPackID, queryLanguage, dbOwner, dbRepo :=
	// 	job.QueryPackId, job.QueryLanguage, job.ORL.Owner, job.ORL.Repo
	queryPackID, dbOwner, dbRepo :=
		job.QueryPackId, job.NWO.Owner, job.NWO.Repo

	serverRoot := os.Getenv("MRVA_SERVER_ROOT")

	// Set up derived paths
	dbPath := filepath.Join(serverRoot, "var/codeql/dbs", dbOwner, dbRepo)
	dbZip := filepath.Join(serverRoot, "codeql/dbs", dbOwner, dbRepo,
		fmt.Sprintf("%s_%s_db.zip", dbOwner, dbRepo))
	dbExtract := filepath.Join(serverRoot, "var/codeql/dbs", dbOwner, dbRepo)

	queryPack := filepath.Join(serverRoot,
		"var/codeql/querypacks", fmt.Sprintf("qp-%d.tgz", queryPackID))
	queryExtract := filepath.Join(serverRoot,
		"var/codeql/querypacks", fmt.Sprintf("qp-%d", queryPackID))

	queryOutDir := filepath.Join(serverRoot,
		"var/codeql/sarif/localrun", dbOwner, dbRepo)
	queryOutFile := filepath.Join(queryOutDir,
		fmt.Sprintf("%s_%s.sarif", dbOwner, dbRepo))

	// Prepare directory, extract database
	if err := os.MkdirAll(dbExtract, 0755); err != nil {
		slog.Error("Failed to create DB directory %s: %v", dbExtract, err)
		return "", err
	}

	if err := utils.UnzipFile(dbZip, dbExtract); err != nil {
		slog.Error("Failed to unzip DB", dbZip, err)
		return "", err
	}

	// Prepare directory, extract query pack
	if err := os.MkdirAll(queryExtract, 0755); err != nil {
		slog.Error("Failed to create query pack directory %s: %v", queryExtract, err)
		return "", err
	}

	if err := utils.UntarGz(queryPack, queryExtract); err != nil {
		slog.Error("Failed to extract querypack %s: %v", queryPack, err)
		return "", err
	}

	// Prepare query result directory
	if err := os.MkdirAll(queryOutDir, 0755); err != nil {
		slog.Error("Failed to create query result directory %s: %v", queryOutDir, err)
		return "", err
	}

	// Run database analyze
	cmd := exec.Command("codeql", "database", "analyze",
		"--format=sarif-latest", "--rerun", "--output", queryOutFile,
		"-j8", dbPath, queryExtract)
	cmd.Dir = serverRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// XX: via message?
		// 	storage.SetStatus(job.QueryPackId, job.ORepo, common.StatusError)
		return "", err
	}

	// Return result path
	return queryOutFile, nil
}

// RunWorker runs a worker that processes jobs from queue
func RunWorker(ctx context.Context, stopChan chan struct{}, queue queue.Queue, wg *sync.WaitGroup) {
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
				result, err := RunAnalysisJob(job)
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

// RunAnalysisJob runs a CodeQL analysis job (AnalyzeJob) returning an AnalyzeResult
func RunAnalysisJob(job common.AnalyzeJob) (common.AnalyzeResult, error) {
	var result = common.AnalyzeResult{
		RequestId:      job.RequestId,
		ResultCount:    0,
		ResultLocation: artifactstore.ArtifactLocation{},
		Status:         common.StatusError,
		// TODO add nwo
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
	slog.Debug("Results archive size", slog.Int("size", len(resultsArchive)))

	result = common.AnalyzeResult{
		RequestId:   job.RequestId,
		ResultCount: runResult.ResultCount,
		// TODO fix
		ResultLocation: artifactstore.ArtifactLocation{},
		// "REPLACE_THIS_WITH_STORED_RESULTS_ARCHIVE", // TODO
		Status: common.StatusSuccess,
		// TODO add NWO
	}

	return result, nil
}
