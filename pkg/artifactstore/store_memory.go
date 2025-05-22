package artifactstore

import (
	"fmt"
	"sync"

	"github.com/hohn/mrvacommander/pkg/common"
)

// InMemoryArtifactStore is an in-memory implementation of the ArtifactStore interface
type InMemoryArtifactStore struct {
	mu      sync.Mutex
	packs   map[string][]byte
	results map[string][]byte
}

func NewInMemoryArtifactStore() *InMemoryArtifactStore {
	return &InMemoryArtifactStore{
		packs:   make(map[string][]byte),
		results: make(map[string][]byte),
	}
}

// GetQueryPack retrieves the query pack from the specified location
func (store *InMemoryArtifactStore) GetQueryPack(location ArtifactLocation) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := location.Key
	data, exists := store.packs[key]
	if !exists {
		return nil, fmt.Errorf("query pack not found: %s", key)
	}
	return data, nil
}

// SaveQueryPack saves the query pack using the session ID and returns the artifact location
func (store *InMemoryArtifactStore) SaveQueryPack(sessionId int, data []byte) (ArtifactLocation, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := fmt.Sprintf("%d-packs", sessionId)
	store.packs[key] = data

	location := ArtifactLocation{
		Bucket: AF_BUCKETNAME_PACKS,
		Key:    key,
	}
	return location, nil
}

// GetResult retrieves the result from the specified location
func (store *InMemoryArtifactStore) GetResult(location ArtifactLocation) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := location.Key
	data, exists := store.results[key]
	if !exists {
		return nil, fmt.Errorf("result not found: %s", key)
	}
	return data, nil
}

// GetResultSize retrieves the size of the result from the specified location
func (store *InMemoryArtifactStore) GetResultSize(location ArtifactLocation) (int, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := location.Key
	data, exists := store.results[key]
	if !exists {
		return 0, fmt.Errorf("result not found: %s", key)
	}
	return len(data), nil
}

// SaveResult saves the result using the JobSpec and returns the artifact location
func (store *InMemoryArtifactStore) SaveResult(jobSpec common.JobSpec, data []byte) (ArtifactLocation, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := fmt.Sprintf("%d-results-%s", jobSpec.SessionID, jobSpec.NameWithOwner)
	store.results[key] = data

	location := ArtifactLocation{
		Bucket: AF_BUCKETNAME_RESULTS,
		Key:    key,
	}
	return location, nil
}
