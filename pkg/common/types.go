package common

// NameWithOwner represents a repository name and its owner name.
type NameWithOwner struct {
	Owner string
	Repo  string
}

// Status represents the status of a job.
type Status int

const (
	StatusPending Status = iota
	StatusInProgress
	StatusSucceeded
	StatusFailed
	StatusCanceled
	StatusTimedOut
)

func (s Status) ToExternalString() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusInProgress:
		return "inProgress"
	case StatusSucceeded:
		return "succeeded"
	case StatusFailed:
		return "failed"
	case StatusCanceled:
		return "canceled"
	case StatusTimedOut:
		return "timedOut"
	default:
		return "unknown"
	}
}

type JobSpec struct {
	SessionID int
	NameWithOwner
}

type StatusSummary struct {
	Overall Status
	Counts  map[Status]int
}
