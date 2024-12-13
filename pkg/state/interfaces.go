package state

import (
	"github.com/hohn/mrvacommander/pkg/common"
	"github.com/hohn/mrvacommander/pkg/queue"
)

// StorageInterface defines the methods required for managing storage operations
// related to server state, e.g. job status, results, and artifacts.
type ServerState interface {
	// NextID increments and returns the next unique ID for a session.
	NextID() int

	// GetResult retrieves the analysis result for the specified job.
	GetResult(js common.JobSpec) (queue.AnalyzeResult, error)

	// GetJobSpecByRepoId retrieves the JobSpec for the specified job Repo ID.
	// TODO: fix this hacky logic
	GetJobSpecByRepoId(sessionId int, jobRepoId int) (common.JobSpec, error)

	// SetResult stores the analysis result for the specified session ID and repository.
	SetResult(js common.JobSpec, ar queue.AnalyzeResult)

	// GetJobList retrieves the list of analysis jobs for the specified session ID.
	GetJobList(sessionId int) ([]queue.AnalyzeJob, error)

	// GetJobInfo retrieves the job information for the specified job specification.
	GetJobInfo(js common.JobSpec) (common.JobInfo, error)

	// SetJobInfo stores the job information for the specified job specification.
	SetJobInfo(js common.JobSpec, ji common.JobInfo)

	// GetStatus retrieves the status of a job for the specified session ID and repository.
	GetStatus(js common.JobSpec) (common.Status, error)

	// SetStatus stores the status of a job for the specified session ID and repository.
	SetStatus(js common.JobSpec, status common.Status)

	// AddJob adds an analysis job to the list of jobs for the specified session ID.
	AddJob(job queue.AnalyzeJob)
}
