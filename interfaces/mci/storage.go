package mci

import (
	"github.com/advanced-security/mrvacommander/types/tsto"
	co "github.com/hohn/ghes-mirva-server/common"
)

type Storage interface {
	NextID() int
	SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error)
	FindAvailableDBs(analysisReposRequested []co.OwnerRepo) (not_found_repos []co.OwnerRepo,
		analysisRepos *map[co.OwnerRepo]tsto.DBLocation)
}
