package queue

import (
	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/common"
)

type QueryLanguage string

// AnalyzeJob represents a job specifying a repository and a query pack to analyze it with.
// This is the message format that the agent receives from the queue.
// TODO: make query_pack_location query_pack_url with a presigned URL
type AnalyzeJob struct {
	Spec              common.JobSpec                 // json:"job_spec"
	QueryPackLocation artifactstore.ArtifactLocation // json:"query_pack_location"
	QueryLanguage     QueryLanguage                  // json:"query_language"
}

// AnalyzeResult represents the result of an analysis job.
// This is the message format that the agent sends to the queue.
// Status will only ever be StatusSuccess or StatusError when sent in a result.
// TODO: make result_location result_archive_url with a presigned URL
type AnalyzeResult struct {
	Spec                 common.JobSpec                 // json:"job_spec"
	Status               common.Status                  // json:"status"
	ResultCount          int                            // json:"result_count"
	ResultLocation       artifactstore.ArtifactLocation // json:"result_location"
	SourceLocationPrefix string                         // json:"source_location_prefix"
	DatabaseSHA          string                         // json:"database_sha"
}
