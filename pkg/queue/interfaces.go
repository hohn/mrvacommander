package queue

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/storage"
)

type Queue interface {
	Jobs() chan common.AnalyzeJob
	Results() chan common.AnalyzeResult
	StartAnalyses(analysis_repos *map[common.NameWithOwner]storage.DBLocation,
		session_id int,
		session_language string)
}
