package storage

import "mrvacommander/pkg/common"

type Storage interface {
	NextID() int
	SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error)
	FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (not_found_repos []common.NameWithOwner,
		analysisRepos *map[common.NameWithOwner]DBLocation)
}
