package agent

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/logger"
	"mrvacommander/pkg/qpstore"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/storage"
	"mrvacommander/utils"

	"log/slog"

	"fmt"
	"path/filepath"

	"os"
	"os/exec"
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
	return
}

func (r *RunnerSingle) worker(wid int) {
	var job common.AnalyzeJob

	for {
		job = <-r.queue.Jobs()

		slog.Debug("Picking up job", "job", job, "worker", wid)

		slog.Debug("Analysis: running", "job", job)
		storage.SetStatus(job.QueryPackId, job.ORepo, common.StatusQueued)

		resultFile, err := r.RunAnalysis(job)
		if err != nil {
			continue
		}

		slog.Debug("Analysis run finished", "job", job)

		res := common.AnalyzeResult{
			RunAnalysisSARIF: resultFile,
			RunAnalysisBQRS:  "", // FIXME ?
		}
		r.queue.Results() <- res
		storage.SetStatus(job.QueryPackId, job.ORepo, common.StatusSuccess)
		storage.SetResult(job.QueryPackId, job.ORepo, res)

	}
}

func (r *RunnerSingle) RunAnalysis(job common.AnalyzeJob) (string, error) {
	// TODO Add multi-language tests including queryLanguage
	// queryPackID, queryLanguage, dbOwner, dbRepo :=
	// 	job.QueryPackId, job.QueryLanguage, job.ORL.Owner, job.ORL.Repo
	queryPackID, dbOwner, dbRepo :=
		job.QueryPackId, job.ORepo.Owner, job.ORepo.Repo

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
		slog.Error("codeql database analyze failed:", "error", err, "job", job)
		storage.SetStatus(job.QueryPackId, job.ORepo, common.StatusError)
		return "", err
	}

	// Return result path
	return queryOutFile, nil
}
