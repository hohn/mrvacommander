package artifactstore

import (
	"fmt"
	"sync"
)

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

func (store *InMemoryArtifactStore) GetQueryPack(location ArtifactLocation) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := location.data["key"]
	data, exists := store.packs[key]
	if !exists {
		return nil, fmt.Errorf("query pack not found: %s", key)
	}
	return data, nil
}

func (store *InMemoryArtifactStore) SaveQueryPack(sessionID int, data []byte) (ArtifactLocation, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := fmt.Sprintf("%d.tgz", sessionID)
	store.packs[key] = data

	location := ArtifactLocation{
		data: map[string]string{
			"bucket": "packs",
			"key":    key,
		},
	}
	return location, nil
}

func (store *InMemoryArtifactStore) GetResult(location ArtifactLocation) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := location.data["key"]
	data, exists := store.results[key]
	if !exists {
		return nil, fmt.Errorf("result not found: %s", key)
	}
	return data, nil
}

func (store *InMemoryArtifactStore) SaveResult(sessionID int, data []byte) (ArtifactLocation, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := fmt.Sprintf("%d.tgz", sessionID)
	store.results[key] = data

	location := ArtifactLocation{
		data: map[string]string{
			"bucket": "results",
			"key":    key,
		},
	}
	return location, nil
}
