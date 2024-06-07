package agent

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/logger"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/storage"

	"log/slog"

	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
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

type RunnerVisibles struct {
	Logger logger.Logger
	Queue  queue.Queue
	// TODO extra package for query pack storage
	QueryPackStore storage.Storage
	// TODO extra package for ql db storage
	QLDBStore storage.Storage
}

func (c *RunnerSingle) Setup(st *RunnerVisibles) {
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

	// FIXME Provide this via environment or explicit argument
	gmsRoot := "/Users/hohn/work-gh/mrva/mrvacommander/cmd/server"

	// Set up derived paths
	dbPath := filepath.Join(gmsRoot, "var/codeql/dbs", dbOwner, dbRepo)
	dbZip := filepath.Join(gmsRoot, "codeql/dbs", dbOwner, dbRepo,
		fmt.Sprintf("%s_%s_db.zip", dbOwner, dbRepo))
	dbExtract := filepath.Join(gmsRoot, "var/codeql/dbs", dbOwner, dbRepo)

	queryPack := filepath.Join(gmsRoot,
		"var/codeql/querypacks", fmt.Sprintf("qp-%d.tgz", queryPackID))
	queryExtract := filepath.Join(gmsRoot,
		"var/codeql/querypacks", fmt.Sprintf("qp-%d", queryPackID))

	queryOutDir := filepath.Join(gmsRoot,
		"var/codeql/sarif/localrun", dbOwner, dbRepo)
	queryOutFile := filepath.Join(queryOutDir,
		fmt.Sprintf("%s_%s.sarif", dbOwner, dbRepo))

	// Prepare directory, extract database
	if err := os.MkdirAll(dbExtract, 0755); err != nil {
		slog.Error("Failed to create DB directory %s: %v", dbExtract, err)
		return "", err
	}

	if err := unzipFile(dbZip, dbExtract); err != nil {
		slog.Error("Failed to unzip DB %s: %v", dbZip, err)
		return "", err
	}

	// Prepare directory, extract query pack
	if err := os.MkdirAll(queryExtract, 0755); err != nil {
		slog.Error("Failed to create query pack directory %s: %v", queryExtract, err)
		return "", err
	}

	if err := untarGz(queryPack, queryExtract); err != nil {
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
	cmd.Dir = gmsRoot
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

// unzipFile extracts a zip file to the specified destination
func unzipFile(zipFile, dest string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fPath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// untarGz extracts a tar.gz file to the specified destination.
func untarGz(tarGzFile, dest string) error {
	file, err := os.Open(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return untar(gzr, dest)
}

// untar extracts a tar archive to the specified destination.
func untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fPath := filepath.Join(dest, header.Name)
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}

			outFile.Close()
		}
	}

	return nil
}
