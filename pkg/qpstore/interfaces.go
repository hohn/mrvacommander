package qpstore

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/qldbstore"
)

type Storage interface {
	NextID() int
	SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error)
	FindAvailableDBs(analysisReposRequested []common.OwnerRepo) (not_found_repos []common.OwnerRepo,
		analysisRepos *map[common.OwnerRepo]qldbstore.DBLocation)
}
