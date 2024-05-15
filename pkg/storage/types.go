package storage

import (
	co "github.com/hohn/ghes-mirva-server/common"
)

type Storage interface {
	NextID() int
	SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error)
	FindAvailableDBs(analysisReposRequested []co.OwnerRepo) (not_found_repos []co.OwnerRepo,
		analysisRepos *map[co.OwnerRepo]DBLocation)
}

type DBLocation struct {
	Prefix string
	File   string
}
