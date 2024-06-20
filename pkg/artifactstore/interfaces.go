package artifactstore

type ArtifactLocation struct {
	// Data is a map of key-value pairs that describe the location of the artifact.
	// For example, a simple key-value pair could be "path" -> "/path/to/artifact.tgz".
	// Alternatively, a more complex example could be "bucket" -> "example", "key" -> "UNIQUE_ARTIFACT_IDENTIFIER".

	// Usage:
	// minio:
	// bucket := location.afdata["bucket"]
	// key := location.afdata["key"]
	//
	// file system:
	// "qp-ID" -> "the/full/path"
	// "result-ID.sarif" -> "the/full/path"
	afdata map[string]string
}

func (al *ArtifactLocation) PathFor(p string) string {
	return al.afdata[p]
}

func (al *ArtifactLocation) Add(k, v string) {
	al.afdata[k] = v
}

type ArtifactStore interface {
	// GetQueryPack returns the query pack as a byte slice for the specified location.
	GetQueryPack(location ArtifactLocation) ([]byte, error)

	// Get the querypack reference (a key for containers, a filename for local storage)
	QPKeyFromID(sessionID int) string

	// SaveQueryPack saves the query pack from the provided byte slice and session ID.
	// It returns the location of the saved query pack.
	SaveQueryPack(sessionID int, data []byte) (ArtifactLocation, error)

	// GetResult returns the result archive as a byte slice for the specified location.
	GetResult(location ArtifactLocation) ([]byte, error)

	// SaveResult saves the result archive from the provided byte slice and session ID.
	// It returns the location of the saved result archive.
	SaveResult(sessionID int, data []byte) (ArtifactLocation, error)
}
