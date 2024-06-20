package qldbstore

import (
	"fmt"
	"log/slog"
	"mrvacommander/pkg/common"
	"os"
	"path/filepath"
)

type FilesystemCodeQLDatabaseStore struct {
	basePath string
}

func NewLocalFilesystemCodeQLDatabaseStore(basePath string) *FilesystemCodeQLDatabaseStore {
	return &FilesystemCodeQLDatabaseStore{
		basePath: basePath,
	}
}

func (store *FilesystemCodeQLDatabaseStore) FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (
	notFoundRepos []common.NameWithOwner,
	foundRepos *map[common.NameWithOwner]CodeQLDatabaseLocation) {

	foundReposMap := make(map[common.NameWithOwner]CodeQLDatabaseLocation)
	for _, repo := range analysisReposRequested {
		location, err := store.GetDatabaseLocationByNWO(repo)
		if err != nil {
			notFoundRepos = append(notFoundRepos, repo)
		} else {
			foundReposMap[repo] = location
		}
	}

	return notFoundRepos, &foundReposMap
}

func (store *FilesystemCodeQLDatabaseStore) GetDatabase(location CodeQLDatabaseLocation) ([]byte, error) {
	path, exists := location.data["path"]
	if !exists {
		// TODO These errors are never exposed.  Don't use errors to guide control flow
		return nil, fmt.Errorf("path not specified in location")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (store *FilesystemCodeQLDatabaseStore) GetDatabaseLocationByNWO(nwo common.NameWithOwner) (CodeQLDatabaseLocation, error) {
	/* Sample location:

	root@21dbc6dc0b7c:/mrva/mrvacommander/cmd/server# ls /mrva/mrvacommander/cmd/server/codeql/dbs/
	google  psycopg

	root@21dbc6dc0b7c:/mrva/mrvacommander/cmd/server# ls /mrva/mrvacommander/cmd/server/codeql/dbs/google/flatbuffers/google_flatbuffers_db.zip
	/mrva/mrvacommander/cmd/server/codeql/dbs/google/flatbuffers/google_flatbuffers_db.zip

	*/
	filePath := filepath.Join(store.basePath, nwo.Owner, nwo.Repo, fmt.Sprintf("%s_%s_db.zip", nwo.Owner, nwo.Repo))

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("Database not found", "nwo", nwo)
		// TODO These errors are never exposed.  Don't use errors to guide control flow
		return CodeQLDatabaseLocation{}, fmt.Errorf("database not found for %s", nwo)
	}

	location := CodeQLDatabaseLocation{
		data: map[string]string{
			"path": filePath,
		},
	}

	return location, nil
}
