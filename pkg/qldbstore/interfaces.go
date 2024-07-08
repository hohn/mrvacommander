package qldbstore

import (
	"mrvacommander/pkg/common"
)

type CodeQLDatabaseLocation struct {
	// `data` is a map of key-value pairs that describe the location of the database.
	// For example, a simple key-value pair could be "path" -> "/path/to/database.zip".
	// A more complex implementation could be "bucket" -> "example", "key" -> "unique_identifier".
	// XX: static types
	data map[string]string
}

type Store interface {
	// FindAvailableDBs returns a map of available databases for the requested analysisReposRequested.
	// It also returns a list of repository NWOs that do not have available databases.
	FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (
		notFoundRepos []common.NameWithOwner,
		foundRepos *map[common.NameWithOwner]CodeQLDatabaseLocation)

	// GetDatabase returns the database as a byte slice for the specified repository.
	// A CodeQL database is a zip archive to be processed by the CodeQL CLI.
	GetDatabase(location CodeQLDatabaseLocation) ([]byte, error)

	// GetDatabaseByNWO returns the database location for the specified repository.
	// FindAvailableDBs should be used in lieu of this method for checking database availability.
	GetDatabaseLocationByNWO(nwo common.NameWithOwner) (CodeQLDatabaseLocation, error)
}
