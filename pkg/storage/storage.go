package storage

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sync"

	co "github.com/hohn/ghes-mirva-server/common"
)

var (
	mutex  sync.Mutex
	result map[co.JobSpec]co.AnalyzeResult = make(map[co.JobSpec]co.AnalyzeResult)
)

type StorageSingle struct {
	CurrentID int
}

func (s *StorageSingle) NextID() int {
	s.CurrentID += 1
	return s.CurrentID
}

func (s *StorageSingle) SaveQueryPack(tgz []byte, sessionId int) (string, error) {
	// Save the tar.gz body
	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("No working directory")
		panic(err)
	}

	dirpath := path.Join(cwd, "var", "codeql", "querypacks")
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		slog.Error("Unable to create query pack output directory",
			"dir", dirpath)
		return "", err
	}

	fpath := path.Join(dirpath, fmt.Sprintf("qp-%d.tgz", sessionId))
	err = os.WriteFile(fpath, tgz, 0644)
	if err != nil {
		slog.Error("unable to save querypack body decoding error", "path", fpath)
		return "", err
	} else {
		slog.Info("Query pack saved to ", "path", fpath)
	}

	return fpath, nil
}

//		Determine for which repositories codeql databases are available.
//
//	 Those will be the analysis_repos.  The rest will be skipped.
func (s *StorageSingle) FindAvailableDBs(analysisReposRequested []co.OwnerRepo) (not_found_repos []co.OwnerRepo,
	analysisRepos *map[co.OwnerRepo]DBLocation) {
	slog.Debug("Looking for available CodeQL databases")

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("No working directory")
		return
	}

	analysisRepos = &map[co.OwnerRepo]DBLocation{}

	not_found_repos = []co.OwnerRepo{}

	for _, rep := range analysisReposRequested {
		dbPrefix := filepath.Join(cwd, "codeql", "dbs", rep.Owner, rep.Repo)
		dbName := fmt.Sprintf("%s_%s_db.zip", rep.Owner, rep.Repo)
		dbPath := filepath.Join(dbPrefix, dbName)

		if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
			slog.Info("Database does not exist for repository ", "owner/repo", rep,
				"path", dbPath)
			not_found_repos = append(not_found_repos, rep)
		} else {
			slog.Info("Found database for ", "owner/repo", rep, "path", dbPath)
			(*analysisRepos)[rep] = DBLocation{Prefix: dbPrefix, File: dbName}
		}
	}
	return not_found_repos, analysisRepos
}

func ArtifactURL(js co.JobSpec, vaid int) (string, error) {
	// We're looking for paths like
	// codeql/sarif/google/flatbuffers/google_flatbuffers.sarif

	ar := GetResult(js)

	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("No host name found")
		return "", nil
	}

	zfpath, err := PackageResults(ar, js.OwnerRepo, vaid)
	if err != nil {
		slog.Error("Error packaging results:", "error", err)
		return "", err
	}
	au := fmt.Sprintf("http://%s:8080/download-server/%s", hostname, zfpath)
	return au, nil
}

func GetResult(js co.JobSpec) co.AnalyzeResult {
	mutex.Lock()
	defer mutex.Unlock()
	ar := result[js]
	return ar
}

func PackageResults(ar co.AnalyzeResult, owre co.OwnerRepo, vaid int) (zipPath string, e error) {
	slog.Debug("Readying zip file with .sarif/.bqrs", "analyze-result", ar)

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("No working directory")
		panic(err)
	}

	// Ensure the output directory exists
	dirpath := path.Join(cwd, "var", "codeql", "localrun", "results")
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		slog.Error("Unable to create results output directory",
			"dir", dirpath)
		return "", err
	}

	// Create a new zip file
	zpath := path.Join(dirpath, fmt.Sprintf("results-%s-%s-%d.zip", owre.Owner, owre.Repo, vaid))

	zfile, err := os.Create(zpath)
	if err != nil {
		return "", err
	}
	defer zfile.Close()

	// Create a new zip writer
	zwriter := zip.NewWriter(zfile)
	defer zwriter.Close()

	// Add each result file to the zip archive
	names := []([]string){{ar.RunAnalysisSARIF, "results.sarif"}}
	for _, fpath := range names {
		file, err := os.Open(fpath[0])
		if err != nil {
			return "", err
		}
		defer file.Close()

		// Create a new file in the zip archive with custom name
		// The client is very specific:
		// if zf.Name != "results.sarif" && zf.Name != "results.bqrs" { continue }

		zipEntry, err := zwriter.Create(fpath[1])
		if err != nil {
			return "", err
		}

		// Copy the contents of the file to the zip entry
		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return "", err
		}
	}
	return zpath, nil
}
