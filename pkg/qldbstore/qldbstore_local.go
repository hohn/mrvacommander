package qldbstore

import (
	"fmt"
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
	foundRepos []common.NameWithOwner) {

	for _, repo := range analysisReposRequested {
		// Form the file path
		filePath := filepath.Join(store.basePath,
			fmt.Sprintf("%s/%s/%s_%s_db.zip", repo.Owner, repo.Repo, repo.Owner, repo.Repo))

		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			notFoundRepos = append(notFoundRepos, repo)
		} else {
			foundRepos = append(foundRepos, repo)
		}
	}

	return notFoundRepos, foundRepos
}

func (store *FilesystemCodeQLDatabaseStore) GetDatabase(location common.NameWithOwner) ([]byte, error) {

	// Form the file path
	filePath := filepath.Join(store.basePath,
		fmt.Sprintf("%s/%s/%s_%s_db.zip", location.Owner, location.Repo, location.Owner, location.Repo))

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found for %s", location)
	}

	// Read file and return it as byte slice
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}
