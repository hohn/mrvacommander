package artifactstore

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
