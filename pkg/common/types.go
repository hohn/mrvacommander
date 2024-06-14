package common

// NameWithOwner represents a repository name and its owner name.
type NameWithOwner struct {
	Owner string
	Repo  string
}

// AnalyzeJob represents a job specifying a repository and a query pack to analyze it with.
// This is the message format that the agent receives from the queue.
type AnalyzeJob struct {
	RequestId     int           // json:"request_id"
	QueryPackId   int           // json:"query_pack_id"
	QueryPackURL  string        // json:"query_pack_url"
	QueryLanguage string        // json:"query_language"
	NWO           NameWithOwner // json:"nwo"
}

// AnalyzeResult represents the result of an analysis job.
// This is the message format that the agent sends to the queue.
// Status will only ever be StatusSuccess or StatusError when sent in a result.
type AnalyzeResult struct {
	Status           Status // json:"status"
	RequestId        int    // json:"request_id"
	ResultCount      int    // json:"result_count"
	ResultArchiveURL string // json:"result_archive_url"
}

// Status represents the status of a job.
type Status int

const (
	StatusInProgress = iota
	StatusQueued
	StatusError
	StatusSuccess
	StatusFailed
)

func (s Status) ToExternalString() string {
	switch s {
	case StatusInProgress:
		return "in_progress"
	case StatusQueued:
		return "queued"
	case StatusError:
		return "error"
	case StatusSuccess:
		return "succeeded"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type JobSpec struct {
	JobID int
	NameWithOwner
}
