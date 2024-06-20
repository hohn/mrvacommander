package artifactstore

import (
	"fmt"
	"log/slog"
	"os"
	"path"
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

	key := location.afdata["key"]
	data, exists := store.packs[key]
	if !exists {
		return nil, fmt.Errorf("query pack not found: %s", key)
	}
	return data, nil
}

func (store *InMemoryArtifactStore) SaveQueryPack(sessionID int, data []byte) (ArtifactLocation, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// Form the retrieval information for the querypack
	qpKey := store.QPKeyFromID(sessionID)
	store.packs[qpKey] = data

	location := ArtifactLocation{
		afdata: map[string]string{
			"bucket": "bucket-not-used-inmemory",
			"key":    qpKey,
		},
	}

	// XX: check

	// Actually store the querypack
	dirpath := os.Getenv("MRVA_QP_ROOT")
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		slog.Error("Unable to create query pack output directory",
			"dir", dirpath)
		return ArtifactLocation{}, err
	}
	fpath := path.Join(dirpath, fmt.Sprintf(qpKey, sessionID))
	err := os.WriteFile(fpath, data, 0644)
	if err != nil {
		slog.Error("Unable to save querypack.  Body decoding error", "path", fpath)
		return ArtifactLocation{}, err
	}

	slog.Info("Query pack saved", "path", fpath)
	return location, nil
}

func (store *InMemoryArtifactStore) QPKeyFromID(sessionID int) string {
	qpKey := fmt.Sprintf("qp-%d.tgz", sessionID)
	return qpKey
}

func (store *InMemoryArtifactStore) GetResult(location ArtifactLocation) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := location.afdata["key"]
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
		afdata: map[string]string{
			"bucket": "results",
			"key":    key,
		},
	}
	return location, nil
}
