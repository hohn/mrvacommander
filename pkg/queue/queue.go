package queue

import (
	"log/slog"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/storage"
)

var (
	NumWorkers int
	Jobs       chan common.AnalyzeJob
	Results    chan common.AnalyzeResult
)

func StartAnalyses(analysis_repos *map[common.OwnerRepo]storage.DBLocation, session_id int,
	session_language string) {
	slog.Debug("Queueing codeql database analyze jobs")

	for orl := range *analysis_repos {
		info := common.AnalyzeJob{
			QueryPackId:   session_id,
			QueryLanguage: session_language,

			ORL: orl,
		}
		Jobs <- info
		storage.SetStatus(session_id, orl, common.StatusQueued)
		storage.AddJob(session_id, info)
	}
}
