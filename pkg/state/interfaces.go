package state

import "mrvacommander/pkg/common"

// StorageInterface defines the methods required for managing storage operations
// related to server state, e.g. job status, results, and artifacts.
type ServerState interface {
	// NextID increments and returns the next unique ID for a session.
	NextID() int

	// GetResult retrieves the analysis result for the specified job.
	GetResult(js common.JobSpec) common.AnalyzeResult

	// SetResult stores the analysis result for the specified session ID and repository.
	SetResult(jobID int, nwo common.NameWithOwner, ar common.AnalyzeResult)

	// GetJobList retrieves the list of analysis jobs for the specified session ID.
	GetJobList(jobID int) []common.AnalyzeJob

	// GetJobInfo retrieves the job information for the specified job specification.
	GetJobInfo(js common.JobSpec) common.JobInfo

	// SetJobInfo stores the job information for the specified job specification.
	SetJobInfo(js common.JobSpec, ji common.JobInfo)

	// GetStatus retrieves the status of a job for the specified session ID and repository.
	GetStatus(jobID int, nwo common.NameWithOwner) common.Status

	// ResultAsFile reads and returns the content of a result file from the specified path.
	ResultAsFile(path string) (string, []byte, error)

	// SetStatus stores the status of a job for the specified session ID and repository.
	SetStatus(jobID int, nwo common.NameWithOwner, status common.Status)

	// AddJob adds an analysis job to the list of jobs for the specified session ID.
	AddJob(jobID int, job common.AnalyzeJob)
}
