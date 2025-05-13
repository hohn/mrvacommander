package artifactstore

import (
	"fmt"

	"github.com/hohn/mrvacommander/pkg/common"
)

// Restrict the keys / values for ArtifactLocation and centralize the common ones
// here
var (
	AF_BUCKETNAME_RESULTS = "mrvabucket"
	AF_BUCKETNAME_PACKS   = "mrvabucket"
)

type ArtifactLocation struct {
	Key    string // location in bucket OR full location for file paths
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
