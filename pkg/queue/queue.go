package queue

import (
	"log/slog"
	"mrvacommander/pkg/storage"

	co "github.com/hohn/ghes-mirva-server/common"
)

var (
	NumWorkers int
	Jobs       chan co.AnalyzeJob
	Results    chan co.AnalyzeResult
)

func StartAnalyses(analysis_repos *map[co.OwnerRepo]storage.DBLocation, session_id int,
	session_language string) {
	slog.Debug("Queueing codeql database analyze jobs")

	for orl := range *analysis_repos {
		info := co.AnalyzeJob{
			QueryPackId:   session_id,
			QueryLanguage: session_language,

			ORL: orl,
		}
		Jobs <- info
		storage.SetStatus(session_id, orl, co.StatusQueued)
		storage.AddJob(session_id, info)
	}
}
