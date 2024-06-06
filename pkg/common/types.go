package common

type AnalyzeJob struct {
	MirvaRequestID int

	QueryPackId   int
	QueryLanguage string

	ORepo OwnerRepo
}

type OwnerRepo struct {
	Owner string
	Repo  string
}

type AnalyzeResult struct {
	RunAnalysisSARIF string
	RunAnalysisBQRS  string
}

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
	OwnerRepo
}
