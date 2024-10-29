package artifactstore

import (
	"fmt"
	"mrvacommander/pkg/common"
)

// xx: afl use
// XX: static types: split by type?
// Restrict the keys / values for ArtifactLocation and centralize the common ones
// here
const (
	AF_BUCKETNAME_RESULTS = "results"
	AF_BUCKETNAME_PACKS   = "packs"
)

type ArtifactLocation struct {
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
