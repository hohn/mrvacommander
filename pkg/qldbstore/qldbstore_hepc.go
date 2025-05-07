package qldbstore

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

func (h *HepcStore) fetchViaHTTP() ([]HepcResult, error) {
	url := fmt.Sprintf("%s/index", h.Endpoint)
	resp, err := http.Get(url)
	if err != nil {
		slog.Warn("Error fetching metadata", "err", err)
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
func (h *HepcStore) fetchViaCli() ([]HepcResult, error) {
	outDir := os.Getenv("MRVA_HEPC_OUTDIR")
	toolName := os.Getenv("MRVA_HEPC_TOOL")

	var missing []string

	if outDir == "" {
		slog.Error("Missing required environment variable", "var", "MRVA_HEPC_OUTDIR")
		missing = append(missing, "MRVA_HEPC_OUTDIR")
	}
	if toolName == "" {
		slog.Error("Missing required environment variable", "var", "MRVA_HEPC_TOOL")
		missing = append(missing, "MRVA_HEPC_TOOL")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	// Expand ~ in outDir
	if strings.HasPrefix(outDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Error("Unable to get home directory", "error", err)
			return nil, err
		}
		outDir = filepath.Join(home, outDir[2:])
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		slog.Error("Failed to create output directory", "error", err)
		return nil, err
	}

	jsonPath := filepath.Join(outDir, "spigot-results.json")

	// ----------------------
	// Go version of
	// spigot-cli bulk-download-results    \
	// --tool-name "$TOOL_NAME"                    \
	// --metadata-only all                         \
	// > "$OUT_DIR/spigot-results.json"
	// ----------------------
	outFile, err := os.Create(jsonPath)
	if err != nil {
		slog.Error("Failed to create spigot output file", "error", err)
		return nil, err
	}
	defer outFile.Close()

	cmd := exec.Command(
		"spigot-cli",
		"bulk-download-results",
		"--tool-name", toolName,
		"--metadata-only", "all",
	)
	cmd.Stdout = outFile

	cmd.Stderr = os.Stderr // for error logging

	if err := cmd.Run(); err != nil {
		slog.Error("spigot-cli failed", "error", err)
		return nil, err
	}
	// ----------------------

	// Decode the resulting JSON file
	f, err := os.Open(jsonPath)
	if err != nil {
		slog.Error("Failed to open JSON output", "path", jsonPath, "error", err)
		return nil, fmt.Errorf("failed to open result file: %w", err)
	}
	defer f.Close()

	var results []HepcResult
	decoder := json.NewDecoder(f)
	for {
		var result HepcResult
		if err := decoder.Decode(&result); err == io.EOF {
			break
		} else if err != nil {
			slog.Warn("Error decoding CLI JSON", "error", err)
			return nil, fmt.Errorf("error decoding CLI JSON: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

func (h *HepcStore) fetchMetadata() ([]HepcResult, error) {
	// Get via request or cli?
	hepcDataViaCli := os.Getenv("MRVA_HEPC_DATAVIACLI")
	if hepcDataViaCli == "1" {
		return h.fetchViaCli()
	} else {
		return h.fetchViaHTTP()
	}
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

func extractDatabaseFromTar(tarStream io.Reader) ([]byte, bool, error) {
	gzReader, err := gzip.NewReader(tarStream)
	if err != nil {
		slog.Error("failed to open gzip stream", "error", err)
		return nil, false, fmt.Errorf("failed to open gzip stream: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("failed to read tar entry", "error", err)
			return nil, false, fmt.Errorf("failed to read tar entry: %w", err)
		}

		if hdr.Name == "artifacts/codeql_database.zip" {
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, tarReader); err != nil {
				slog.Error("failed to extract zip from tar", "error", err)
				return nil, false, fmt.Errorf("failed to extract zip from tar: %w", err)
			}
			return buf.Bytes(), true, nil
		}
	}

	return nil, false, nil // not found
}

func (h *HepcStore) GetDatabase(location common.NameWithOwner) ([]byte, error) {
	h.cacheMutex.Lock()
	if time.Since(h.cacheLastUpdated) > h.cacheDuration {
		results, err := h.fetchMetadata()
		if err != nil {
			slog.Error("error refreshing metadata cache", "error", err)
			h.cacheMutex.Unlock()
			return nil, fmt.Errorf("error refreshing metadata cache: %w", err)
		}
		h.metadataCache = results
		h.cacheLastUpdated = time.Now()
	}
	cachedResults := h.metadataCache
	h.cacheMutex.Unlock()

	key := fmt.Sprintf("%s/%s", location.Owner, location.Repo)

	var resultURL string
	for _, result := range cachedResults {
		if result.Projname == key {
			resultURL = result.ResultURL
			break
		}
	}

	if resultURL == "" {
		slog.Error("database not found in metadata", "repo", key)
		return nil, fmt.Errorf("database not found for repository: %s", key)
	}

	resp, err := http.Get(replaceHepcURL(resultURL))
	if err != nil {
		slog.Error("failed to fetch database", "url", resultURL, "error", err)
		return nil, fmt.Errorf("error fetching database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("non-OK HTTP status", "status", resp.Status, "url", resultURL)
		return nil, fmt.Errorf("non-OK HTTP status for database fetch: %s", resp.Status)
	}

	// Buffer the full gzip tar stream into RAM
	fullBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("error reading full database stream into memory", "error", err)
		return nil, fmt.Errorf("error reading database content: %w", err)
	}

	// Create a fresh reader from RAM buffer for extraction
	data, found, err := extractDatabaseFromTar(bytes.NewReader(fullBody))
	if err != nil {
		slog.Error("error extracting from tar stream", "error", err)
		return nil, err
	}

	if found {
		slog.Info("found nested zip", "path", "artifacts/codeql_database.zip")
		return data, nil
	}

	slog.Info("nested zip not found, returning full original stream from buffer")
	return fullBody, nil
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
