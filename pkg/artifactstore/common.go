package artifactstore

import (
	"fmt"
	"mrvacommander/pkg/common"
)

// XX: static types: split by type?
// Restrict the keys / values for ArtifactLocation and centralize the common ones
// here
const (
	AF_VAL_BUCKET_RESULTS = "results"
	AF_VAL_BUCKET_PACKS   = "packs"
	AF_KEY_BUCKET         = "bucket"
	AF_KEY_KEY            = "key"
)

type ArtifactLocation struct {
	// Data is a map of key-value pairs that describe the location of the artifact.
	// For example, a simple key-value pair could be "path" -> "/path/to/artifact.tgz".
	// Alternatively, a more complex example could be "bucket" -> "example", "key" -> "UNIQUE_ARTIFACT_IDENTIFIER".
	// XX: static types
	// Data   map[string]string `json:"data"`
	Key    string // location in bucket
	Bucket string // which bucket: packs or results
}

// deriveKeyFromSessionId generates a key for a query pack based on the job ID
func deriveKeyFromSessionId(sessionId int) string {
	return fmt.Sprintf("%d", sessionId)
}

// deriveKeyFromJobSpec generates a key for a result based on the JobSpec
func deriveKeyFromJobSpec(jobSpec common.JobSpec) string {
	return fmt.Sprintf("%d-%s", jobSpec.SessionID, jobSpec.NameWithOwner)
}
