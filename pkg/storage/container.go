package storage

import (
	"fmt"
	"log/slog"
	"mrvacommander/pkg/common"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBmutex sync.Mutex
)

func (s *StorageContainer) NextID() int {
	// TODO update via db
	return 12345
}

func (s *StorageContainer) SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error) {
	// TODO save and return path
	return "todo:no-path-yet", nil
}

func (s *StorageContainer) FindAvailableDBs(analysisReposRequested []common.OwnerRepo) (notFoundRepos []common.OwnerRepo, analysisRepos *map[common.OwnerRepo]DBLocation) {
	// TODO  s.FindAvailableDBs() via postgres
	analysisRepos = &map[common.OwnerRepo]DBLocation{}
	notFoundRepos = []common.OwnerRepo{}

	return notFoundRepos, analysisRepos
}

func NewStorageContainer(startingID int) (*StorageContainer, error) {
	// Set up the database connection string
	const (
		host     = "postgres"
		port     = 5432
		user     = "exampleuser"
		password = "examplepass"
		dbname   = "exampledb"
	)

	// Open the database connection
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Error connecting to the database", "err", err)
		return nil, err
	}

	// Check and set up the database
	s := StorageContainer{RequestID: startingID, DB: db}
	if s.hasTables() {
		s.loadState()
	} else {
		if err = s.setupDB(); err != nil {
			return nil, err
		}
		s.setFresh()
	}

	return &s, nil
}

func (s *StorageContainer) setFresh() {
	// TODO Set initial state
}

func (s *StorageContainer) setupDB() error {
	// TODO Migrate the schemas
	msg := "Failed to initialize database "

	if err := s.DB.AutoMigrate(&DBInfo{}); err != nil {
		slog.Error(msg, "table", "dbinfo")
		return err
	}
	if err := s.DB.AutoMigrate(&DBJobs{}); err != nil {
		slog.Error(msg, "table", "dbjobs")
		return err
	}
	if err := s.DB.AutoMigrate(&DBResult{}); err != nil {
		slog.Error(msg, "table", "dbresult")
		return err
	}
	if err := s.DB.AutoMigrate(&DBStatus{}); err != nil {
		slog.Error(msg, "table", "dbstatus")
		return err
	}

	return nil
}

func (s *StorageContainer) loadState() {
	// TODO load the state
	return
}

func (s *StorageContainer) hasTables() bool {
	// TODO sql query to check for tables
	return false
}

// ================ TODO migrate

// func (s *StorageSingle) NextID() int {
// 	s.RequestID += 1
// 	return s.RequestID
// }

// func (s *StorageSingle) SaveQueryPack(tgz []byte, sessionId int) (string, error) {
// 	// Save the tar.gz body
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		slog.Error("No working directory")
// 		panic(err)
// 	}

// 	dirpath := path.Join(cwd, "var", "codeql", "querypacks")
// 	if err := os.MkdirAll(dirpath, 0755); err != nil {
// 		slog.Error("Unable to create query pack output directory",
// 			"dir", dirpath)
// 		return "", err
// 	}

// 	fpath := path.Join(dirpath, fmt.Sprintf("qp-%d.tgz", sessionId))
// 	err = os.WriteFile(fpath, tgz, 0644)
// 	if err != nil {
// 		slog.Error("unable to save querypack body decoding error", "path", fpath)
// 		return "", err
// 	} else {
// 		slog.Info("Query pack saved to ", "path", fpath)
// 	}

// 	return fpath, nil
// }

// //		Determine for which repositories codeql databases are available.
// //
// //	 Those will be the analysis_repos.  The rest will be skipped.
// func (s *StorageSingle) FindAvailableDBs(analysisReposRequested []common.OwnerRepo) (notFoundRepos []common.OwnerRepo,
// 	analysisRepos *map[common.OwnerRepo]DBLocation) {
// 	slog.Debug("Looking for available CodeQL databases")

// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		slog.Error("No working directory")
// 		return
// 	}

// 	analysisRepos = &map[common.OwnerRepo]DBLocation{}

// 	notFoundRepos = []common.OwnerRepo{}

// 	for _, rep := range analysisReposRequested {
// 		dbPrefix := filepath.Join(cwd, "codeql", "dbs", rep.Owner, rep.Repo)
// 		dbName := fmt.Sprintf("%s_%s_db.zip", rep.Owner, rep.Repo)
// 		dbPath := filepath.Join(dbPrefix, dbName)

// 		if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
// 			slog.Info("Database does not exist for repository ", "owner/repo", rep,
// 				"path", dbPath)
// 			notFoundRepos = append(notFoundRepos, rep)
// 		} else {
// 			slog.Info("Found database for ", "owner/repo", rep, "path", dbPath)
// 			(*analysisRepos)[rep] = DBLocation{Prefix: dbPrefix, File: dbName}
// 		}
// 	}
// 	return notFoundRepos, analysisRepos
// }

// func ArtifactURL(js common.JobSpec, vaid int) (string, error) {
// 	// We're looking for paths like
// 	// codeql/sarif/google/flatbuffers/google_flatbuffers.sarif

// 	ar := GetResult(js)

// 	hostname, err := os.Hostname()
// 	if err != nil {
// 		slog.Error("No host name found")
// 		return "", nil
// 	}

// 	zfpath, err := PackageResults(ar, js.OwnerRepo, vaid)
// 	if err != nil {
// 		slog.Error("Error packaging results:", "error", err)
// 		return "", err
// 	}
// 	au := fmt.Sprintf("http://%s:8080/download-server/%s", hostname, zfpath)
// 	return au, nil
// }

// func GetResult(js common.JobSpec) common.AnalyzeResult {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	ar := result[js]
// 	return ar
// }

// func SetResult(sessionid int, orl common.OwnerRepo, ar common.AnalyzeResult) {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	result[common.JobSpec{RequestID: sessionid, OwnerRepo: orl}] = ar
// }

// func PackageResults(ar common.AnalyzeResult, owre common.OwnerRepo, vaid int) (zipPath string, e error) {
// 	slog.Debug("Readying zip file with .sarif/.bqrs", "analyze-result", ar)

// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		slog.Error("No working directory")
// 		panic(err)
// 	}

// 	// Ensure the output directory exists
// 	dirpath := path.Join(cwd, "var", "codeql", "localrun", "results")
// 	if err := os.MkdirAll(dirpath, 0755); err != nil {
// 		slog.Error("Unable to create results output directory",
// 			"dir", dirpath)
// 		return "", err
// 	}

// 	// Create a new zip file
// 	zpath := path.Join(dirpath, fmt.Sprintf("results-%s-%s-%d.zip", owre.Owner, owre.Repo, vaid))

// 	zfile, err := os.Create(zpath)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer zfile.Close()

// 	// Create a new zip writer
// 	zwriter := zip.NewWriter(zfile)
// 	defer zwriter.Close()

// 	// Add each result file to the zip archive
// 	names := []([]string){{ar.RunAnalysisSARIF, "results.sarif"}}
// 	for _, fpath := range names {
// 		file, err := os.Open(fpath[0])
// 		if err != nil {
// 			return "", err
// 		}
// 		defer file.Close()

// 		// Create a new file in the zip archive with custom name
// 		// The client is very specific:
// 		// if zf.Name != "results.sarif" && zf.Name != "results.bqrs" { continue }

// 		zipEntry, err := zwriter.Create(fpath[1])
// 		if err != nil {
// 			return "", err
// 		}

// 		// Copy the contents of the file to the zip entry
// 		_, err = io.Copy(zipEntry, file)
// 		if err != nil {
// 			return "", err
// 		}
// 	}
// 	return zpath, nil
// }

// func GetJobList(sessionid int) []common.AnalyzeJob {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	return jobs[sessionid]
// }

// func GetJobInfo(js common.JobSpec) common.JobInfo {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	return info[js]
// }

// func SetJobInfo(js common.JobSpec, ji common.JobInfo) {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	info[js] = ji
// }

// func GetStatus(sessionid int, orl common.OwnerRepo) common.Status {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	return status[common.JobSpec{RequestID: sessionid, OwnerRepo: orl}]
// }

// func ResultAsFile(path string) (string, []byte, error) {
// 	fpath := path
// 	if !filepath.IsAbs(path) {
// 		fpath = "/" + path
// 	}

// 	file, err := os.ReadFile(fpath)
// 	if err != nil {
// 		slog.Warn("Failed to read results file", fpath, err)
// 		return "", nil, err
// 	}

// 	return fpath, file, nil
// }

// func SetStatus(sessionid int, orl common.OwnerRepo, s common.Status) {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	status[common.JobSpec{RequestID: sessionid, OwnerRepo: orl}] = s
// }

// func AddJob(sessionid int, job common.AnalyzeJob) {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	jobs[sessionid] = append(jobs[sessionid], job)
// }
