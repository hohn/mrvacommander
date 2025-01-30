package qldbstore

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hohn/mrvacommander/pkg/common"
)

const defaultCacheDurationMinutes = 60

type HepcStore struct {
	Endpoint         string
	metadataCache    []HepcResult
	cacheLastUpdated time.Time
	cacheMutex       sync.Mutex
	cacheDuration    time.Duration
}

type HepcResult struct {
	GitBranch         string `json:"git_branch"`
	GitCommitID       string `json:"git_commit_id"`
	GitRepo           string `json:"git_repo"`
	IngestionDatetime string `json:"ingestion_datetime_utc"`
	ResultURL         string `json:"result_url"`
	ToolID            string `json:"tool_id"`
	ToolName          string `json:"tool_name"`
	ToolVersion       string `json:"tool_version"`
	Projname          string `json:"projname"`
}

func NewHepcStore(endpoint string) *HepcStore {
	cacheDuration := getMetaCacheDuration()
	return &HepcStore{
		Endpoint:      endpoint,
		cacheDuration: cacheDuration,
	}
}

func getMetaCacheDuration() time.Duration {
	durationStr := os.Getenv("MRVA_HEPC_CACHE_DURATION")
	if durationStr == "" {
		return time.Minute * defaultCacheDurationMinutes
	}
	duration, err := strconv.Atoi(durationStr)
	if err != nil {
		slog.Warn("Invalid MRVA_HEPC_CACHE_DURATION value. Using default",
			durationStr, defaultCacheDurationMinutes,
		)
		return time.Minute * defaultCacheDurationMinutes
	}
	return time.Minute * time.Duration(duration)
}

func (h *HepcStore) fetchMetadata() ([]HepcResult, error) {
	url := fmt.Sprintf("%s/index", h.Endpoint)
	resp, err := http.Get(url)
	if err != nil {
		slog.Warn("Error fetching metadata.", err)
		return nil, fmt.Errorf("error fetching metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Non-OK HTTP status", resp.Status)
		return nil, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	var results []HepcResult
	decoder := json.NewDecoder(resp.Body)
	for {
		var result HepcResult
		if err := decoder.Decode(&result); err == io.EOF {
			break
		} else if err != nil {
			slog.Warn("Error decoding JSON", err)
			return nil, fmt.Errorf("error decoding JSON: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

func (h *HepcStore) FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (
	notFoundRepos []common.NameWithOwner,
	foundRepos []common.NameWithOwner) {

	// Check cache
	h.cacheMutex.Lock()
	if time.Since(h.cacheLastUpdated) > h.cacheDuration {
		// Cache is expired or not set; refresh
		results, err := h.fetchMetadata()
		if err != nil {
			h.cacheMutex.Unlock()
			slog.Warn("Error fetching metadata", err)
			return analysisReposRequested, nil
		}
		h.metadataCache = results
		h.cacheLastUpdated = time.Now()
	}
	cachedResults := h.metadataCache
	h.cacheMutex.Unlock()

	// Compare against requested repos
	repoSet := make(map[string]struct{})
	for _, result := range cachedResults {
		repoSet[result.Projname] = struct{}{}
	}

	for _, reqRepo := range analysisReposRequested {
		repoKey := fmt.Sprintf("%s/%s", reqRepo.Owner, reqRepo.Repo)
		if _, exists := repoSet[repoKey]; exists {
			foundRepos = append(foundRepos, reqRepo)
		} else {
			notFoundRepos = append(notFoundRepos, reqRepo)
		}
	}

	return notFoundRepos, foundRepos
}

func (h *HepcStore) GetDatabase(location common.NameWithOwner) ([]byte, error) {
	// Ensure metadata is up-to-date by using the cache
	h.cacheMutex.Lock()
	if time.Since(h.cacheLastUpdated) > h.cacheDuration {
		// Refresh the metadata cache if it is stale
		results, err := h.fetchMetadata()
		if err != nil {
			h.cacheMutex.Unlock()
			return nil, fmt.Errorf("error refreshing metadata cache: %w", err)
		}
		h.metadataCache = results
		h.cacheLastUpdated = time.Now()
	}
	cachedResults := h.metadataCache
	h.cacheMutex.Unlock()

	// Construct the key for the requested database
	key := fmt.Sprintf("%s/%s", location.Owner, location.Repo)

	// Locate the result URL in the cached metadata
	var resultURL string
	for _, result := range cachedResults {
		if result.Projname == key {
			resultURL = result.ResultURL
			break
		}
	}

	if resultURL == "" {
		return nil, fmt.Errorf("database not found for repository: %s", key)
	}

	// Fetch the database content
	resp, err := http.Get(replaceHepcURL(resultURL))
	if err != nil {
		return nil, fmt.Errorf("error fetching database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-OK HTTP status for database fetch: %s", resp.Status)
	}

	// Read and return the database data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading database content: %w", err)
	}

	return data, nil
}

// replaceHepcURL replaces the fixed "http://hepc" with the value from
// MRVA_HEPC_ENDPOINT
func replaceHepcURL(originalURL string) string {
	hepcEndpoint := os.Getenv("MRVA_HEPC_ENDPOINT")
	if hepcEndpoint == "" {
		hepcEndpoint = "http://hepc:8070" // Default fallback
	}

	// Replace "http://hepc" at the beginning of the URL
	newURL := strings.Replace(originalURL, "http://hepc", hepcEndpoint, 1)

	return newURL
}
