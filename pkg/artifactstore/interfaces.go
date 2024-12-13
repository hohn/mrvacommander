package artifactstore

import "github.com/hohn/mrvacommander/pkg/common"

type Store interface {
	// GetQueryPack retrieves the query pack from the specified location.
	GetQueryPack(location ArtifactLocation) ([]byte, error)

	// SaveQueryPack saves the query pack using the session ID and returns the artifact location.
	SaveQueryPack(sessionId int, data []byte) (ArtifactLocation, error)

	// GetResult retrieves the result from the specified location.
	GetResult(location ArtifactLocation) ([]byte, error)

	// GetResultSize retrieves the size of the result from the specified location.
	GetResultSize(location ArtifactLocation) (int, error)

	// SaveResult saves the result using the JobSpec and returns the artifact location.
	SaveResult(jobSpec common.JobSpec, data []byte) (ArtifactLocation, error)
}
