package artifactstore

import (
	"fmt"
	"mrvacommander/pkg/common"
	"sync"
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

	key := location.Data["key"]
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

	key := deriveKeyFromSessionId(sessionId)
	store.packs[key] = data

	location := ArtifactLocation{
		Data: map[string]string{
			"bucket": "packs",
			"key":    key,
		},
	}
	return location, nil
}

// GetResult retrieves the result from the specified location
func (store *InMemoryArtifactStore) GetResult(location ArtifactLocation) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := location.Data["key"]
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

	key := location.Data["key"]
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

	key := deriveKeyFromJobSpec(jobSpec)
	store.results[key] = data

	location := ArtifactLocation{
		Data: map[string]string{
			"bucket": "results",
			"key":    key,
		},
	}
	return location, nil
}
