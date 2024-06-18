package artifactstore

type ArtifactLocation struct {
	// Data is a map of key-value pairs that describe the location of the artifact.
	// For example, a simple key-value pair could be "path" -> "/path/to/artifact.tgz".
	// Alternatively, a more complex example could be "bucket" -> "example", "key" -> "UNIQUE_ARTIFACT_IDENTIFIER".
	data map[string]string
}

type ArtifactStore interface {
	// GetQueryPack returns the query pack as a byte slice for the specified location.
	GetQueryPack(location ArtifactLocation) ([]byte, error)

	// SaveQueryPack saves the query pack from the provided byte slice and session ID.
	// It returns the location of the saved query pack.
	SaveQueryPack(sessionID int, data []byte) (ArtifactLocation, error)

	// GetResult returns the result archive as a byte slice for the specified location.
	GetResult(location ArtifactLocation) ([]byte, error)

	// SaveResult saves the result archive from the provided byte slice and session ID.
	// It returns the location of the saved result archive.
	SaveResult(sessionID int, data []byte) (ArtifactLocation, error)
}
