package common

// NameWithOwner represents a repository name and its owner name.
type NameWithOwner struct {
	Owner string
	Repo  string
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
	SessionID int
	NameWithOwner
}
