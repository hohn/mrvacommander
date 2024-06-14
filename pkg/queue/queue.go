package queue

import (
	"log/slog"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/storage"
)

func (q *QueueSingle) Jobs() chan common.AnalyzeJob {
	return q.jobs
}

func (q *QueueSingle) Results() chan common.AnalyzeResult {
	return q.results
}

func (q *QueueSingle) StartAnalyses(analysis_repos *map[common.NameWithOwner]storage.DBLocation, session_id int,
	session_language string) {
	slog.Debug("Queueing codeql database analyze jobs")

	for nwo := range *analysis_repos {
		info := common.AnalyzeJob{
			QueryPackId:   session_id,
			QueryLanguage: session_language,
			NWO:           nwo,
		}
		q.jobs <- info
		storage.SetStatus(session_id, nwo, common.StatusQueued)
		storage.AddJob(session_id, info)
	}
}
