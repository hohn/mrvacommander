package qldbstore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hohn/mrvacommander/pkg/common"
)

type HepcStore struct {
	Endpoint string
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
	return &HepcStore{Endpoint: endpoint}
}

func (h *HepcStore) FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (
	notFoundRepos []common.NameWithOwner,
	foundRepos []common.NameWithOwner) {

	// Fetch the metadata.json from the Hepc server
	url := fmt.Sprintf("%s/index", h.Endpoint)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching metadata: %v\n", err)
		return analysisReposRequested, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Non-OK HTTP status: %s\n", resp.Status)
		return analysisReposRequested, nil
	}

	// Decode the response
	var results []HepcResult
	decoder := json.NewDecoder(resp.Body)
	for {
		var result HepcResult
		if err := decoder.Decode(&result); err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("Error decoding JSON: %v\n", err)
			return analysisReposRequested, nil
		}
		results = append(results, result)
	}

	// Compare against requested repos
	repoSet := make(map[string]struct{})
	for _, result := range results {
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
	// Fetch the latest results for the specified repository
	url := fmt.Sprintf("%s/api/v1/latest_results/codeql-all", h.Endpoint)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching database metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	var latestResults []HepcResult
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&latestResults); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	// Find the correct result for the requested repo
	repoKey := fmt.Sprintf("%s/%s", location.Owner, location.Repo)
	for _, result := range latestResults {
		if result.Projname == repoKey {
			// Fetch the database as a byte slice
			dbResp, err := http.Get(result.ResultURL)
			if err != nil {
				return nil, fmt.Errorf("error fetching database: %w", err)
			}
			defer dbResp.Body.Close()

			if dbResp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("non-OK HTTP status for database fetch: %s", dbResp.Status)
			}

			return io.ReadAll(dbResp.Body)
		}
	}

	return nil, fmt.Errorf("database not found for repository: %s", repoKey)
}
