package storage

import (
	"mrvacommander/pkg/common"
)

type Storage interface {
	NextID() int
	SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error)
	FindAvailableDBs(analysisReposRequested []common.OwnerRepo) (not_found_repos []common.OwnerRepo,
		analysisRepos *map[common.OwnerRepo]DBLocation)
}

type DBLocation struct {
	Prefix string
	File   string
}
