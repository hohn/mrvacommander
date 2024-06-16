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

	"mrvacommander/pkg/common"
)

var (
	jobs   map[int][]common.AnalyzeJob             = make(map[int][]common.AnalyzeJob)
	info   map[common.JobSpec]common.JobInfo       = make(map[common.JobSpec]common.JobInfo)
	status map[common.JobSpec]common.Status        = make(map[common.JobSpec]common.Status)
	result map[common.JobSpec]common.AnalyzeResult = make(map[common.JobSpec]common.AnalyzeResult)
	mutex  sync.Mutex
)

func NewStorageSingle(startingID int, v *Visibles) *StorageSingle {
	s := StorageSingle{currentID: startingID}

	s.modules = v

	return &s
}

func (s *StorageSingle) NextID() int {
	s.currentID += 1
	return s.currentID
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
func (s *StorageSingle) FindAvailableDBs(analysisReposRequested []common.OwnerRepo) (not_found_repos []common.OwnerRepo,
	analysisRepos *map[common.OwnerRepo]DBLocation) {
	slog.Debug("Looking for available CodeQL databases")

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("No working directory")
		return
	}

	analysisRepos = &map[common.OwnerRepo]DBLocation{}

	not_found_repos = []common.OwnerRepo{}

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

func ArtifactURL(js common.JobSpec, vaid int) (string, error) {
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
	// TODO Need url valid in container network and externally
	// For now, we assume the container port 8080 is port 8080 on user's machine
	hostname = "localhost"
	au := fmt.Sprintf("http://%s:8080/download-server/%s", hostname, zfpath)
	return au, nil
}

func GetResult(js common.JobSpec) common.AnalyzeResult {
	mutex.Lock()
	defer mutex.Unlock()
	ar := result[js]
	return ar
}

func SetResult(sessionid int, orl common.OwnerRepo, ar common.AnalyzeResult) {
	mutex.Lock()
	defer mutex.Unlock()
	result[common.JobSpec{JobID: sessionid, OwnerRepo: orl}] = ar
}

func PackageResults(ar common.AnalyzeResult, owre common.OwnerRepo, vaid int) (zipPath string, e error) {
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

func GetJobList(sessionid int) []common.AnalyzeJob {
	mutex.Lock()
	defer mutex.Unlock()
	return jobs[sessionid]
}

func GetJobInfo(js common.JobSpec) common.JobInfo {
	mutex.Lock()
	defer mutex.Unlock()
	return info[js]
}

func SetJobInfo(js common.JobSpec, ji common.JobInfo) {
	mutex.Lock()
	defer mutex.Unlock()
	info[js] = ji
}

func GetStatus(sessionid int, orl common.OwnerRepo) common.Status {
	mutex.Lock()
	defer mutex.Unlock()
	return status[common.JobSpec{JobID: sessionid, OwnerRepo: orl}]
}

func ResultAsFile(path string) (string, []byte, error) {
	fpath := path
	if !filepath.IsAbs(path) {
		fpath = "/" + path
	}

	file, err := os.ReadFile(fpath)
	if err != nil {
		slog.Warn("Failed to read results file", fpath, err)
		return "", nil, err
	}

	return fpath, file, nil
}

func SetStatus(sessionid int, orl common.OwnerRepo, s common.Status) {
	mutex.Lock()
	defer mutex.Unlock()
	status[common.JobSpec{JobID: sessionid, OwnerRepo: orl}] = s
}

func AddJob(sessionid int, job common.AnalyzeJob) {
	mutex.Lock()
	defer mutex.Unlock()
	jobs[sessionid] = append(jobs[sessionid], job)
}
