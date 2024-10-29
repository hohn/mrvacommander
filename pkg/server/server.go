package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/queue"
	"mrvacommander/utils"

	"github.com/gorilla/mux"
)

func (c *CommanderSingle) startAnalyses(
	analysisRepos []common.NameWithOwner,
	queryPackLocation artifactstore.ArtifactLocation,
	sessionId int,
	queryLanguage queue.QueryLanguage) {

	slog.Debug("Queueing analysis jobs", "count", len(analysisRepos))

	for _, nwo := range analysisRepos {
		jobSpec := common.JobSpec{
			SessionID:     sessionId,
			NameWithOwner: nwo,
		}
		info := queue.AnalyzeJob{
			Spec:              jobSpec,
			QueryPackLocation: queryPackLocation,
			QueryLanguage:     queryLanguage,
		}
		c.v.Queue.Jobs() <- info
		c.v.State.SetStatus(jobSpec, common.StatusQueued)
		c.v.State.AddJob(info)
	}
}

func setupEndpoints(c CommanderAPI) {
	r := mux.NewRouter()

	// Root handler
	r.HandleFunc("/", c.RootHandler)

	// Endpoints for submitting new analyses
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses", c.MRVARequest)
	r.HandleFunc("/repositories/{controller_repo_id}/code-scanning/codeql/variant-analyses", c.MRVARequestID)

	// Endpoints for status requests
	// This is also the first request made when downloading; the difference is in the client-side handling.
	// TODO: better document / standardize this: {codeql_variant_analysis_id} is the session ID
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}", c.MRVAStatus)
	r.HandleFunc("/repositories/{controller_repo_id}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}", c.MRVAStatusID)

	// XX: Handle endpoint
	//			  /repos/tdlib/telegram-bot-apictsj8529d9/code-scanning/codeql/databases/cpp
	// Endpoints for getting a URL to download artifacts
	//			  /repos/tdlib	     /telegram.../code-scanning/codeql/databases/cpp
	r.HandleFunc("/repos/{repo_owner}/{repo_name}/code-scanning/codeql/databases/{repo_language}", c.MRVADownloadQLDB)
	r.HandleFunc("/repos/{controller_owner}/{controller_repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}/repos/{repo_owner}/{repo_name}", c.MRVADownloadArtifact)
	r.HandleFunc("/repositories/{controller_repo_id}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}/repositories/{repository_id}", c.MRVADownloadArtifactID)

	// Endpoint to serve downloads using encoded JobSpec
	r.HandleFunc("/download/{encoded_job_spec}", c.MRVADownloadServe)

	// Handler for unhandled endpoints
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Error("Unhandled endpoint", "method", r.Method, "uri", r.RequestURI)
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	go ListenAndServe(r)
}

func ListenAndServe(r *mux.Router) {
	// Bind to a port and pass our router in
	// The port is configurable via environment variable or default to 8080
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		slog.Error("Error starting server:", "error", err)
		os.Exit(1)
	}
}

// TODO: check the caller as well so that it still returns statuses if no jobs exist (e.g. missing dbs) --
func (c *CommanderSingle) submitEmptyStatusResponse(w http.ResponseWriter,
	jsSessionID int,
) {
	slog.Debug("Submitting status response for empty job list", "job_id", jsSessionID)

	// TODO Can/need this struct contain more info when |jobs| == 0?
	ji := common.JobInfo{
		QueryLanguage: "",
		CreatedAt:     "",
		UpdatedAt:     "",

		SkippedRepositories: common.SkippedRepositories{
			AccessMismatchRepos: common.AccessMismatchRepos{
				RepositoryCount: 0,
				Repositories:    []common.Repository{},
			},
			NotFoundRepos: common.NotFoundRepos{
				RepositoryCount:     0,
				RepositoryFullNames: []string{},
			},
			NoCodeqlDBRepos: common.NoCodeqlDBRepos{
				RepositoryCount: 0,
				Repositories:    []common.Repository{},
			},
			OverLimitRepos: common.OverLimitRepos{
				RepositoryCount: 0,
				Repositories:    []common.Repository{},
			},
		},
	}

	scannedRepos := []common.ScannedRepo{}

	var jobStatus common.Status
	jobStatus = common.StatusSuccess

	status := common.StatusResponse{
		SessionId:            jsSessionID,
		ControllerRepo:       common.ControllerRepo{},
		Actor:                common.Actor{},
		QueryLanguage:        ji.QueryLanguage,
		QueryPackURL:         "", // FIXME
		CreatedAt:            ji.CreatedAt,
		UpdatedAt:            ji.UpdatedAt,
		ActionsWorkflowRunID: -1, // FIXME
		Status:               jobStatus.ToExternalString(),
		ScannedRepositories:  scannedRepos,
		SkippedRepositories:  ji.SkippedRepositories,
	}

	// Encode the response as JSON
	submitStatus, err := json.Marshal(status)
	if err != nil {
		slog.Error("Error encoding response as JSON:",
			"error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send analysisReposJSON via ResponseWriter
	w.Header().Set("Content-Type", "application/json")
	w.Write(submitStatus)
}

// TODO: fix this so that it can return partial results?? if possible?
func (c *CommanderSingle) submitStatusResponse(w http.ResponseWriter, js common.JobSpec, ji common.JobInfo) {
	slog.Debug("Submitting status response", "job_id", js.SessionID)

	scannedRepos := []common.ScannedRepo{}

	jobs, err := c.v.State.GetJobList(js.SessionID)
	if err != nil {
		slog.Error("Error getting job list", "error", err.Error())
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Loop through all jobs under the same session id
	// TODO: as a high priority, fix this hacky job IDing by index
	// this may break with other state implementations
	for jobRepoId, job := range jobs {
		// Get the job status
		status, err := c.v.State.GetStatus(job.Spec)
		if err != nil {
			slog.Error("Error getting status", "error", err.Error())
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		// Get the job result if complete, otherwise return default values
		var artifactSize int
		var resultCount int

		if status != common.StatusSuccess {
			// If the job is not successful, we don't need to get the result
			artifactSize = 0
			resultCount = 0
		} else {
			jobResult, err := c.v.State.GetResult(job.Spec)
			if err != nil {
				slog.Error("Error getting result", "error", err.Error())
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			artifactSize, err = c.v.Artifacts.GetResultSize(jobResult.ResultLocation)
			if err != nil {
				slog.Error("Error getting artifact size", "error", err.Error())
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			resultCount = jobResult.ResultCount
		}

		// Append all scanned (complete and incomplete) repos to the response
		scannedRepos = append(scannedRepos,
			common.ScannedRepo{
				Repository: common.Repository{
					ID:              jobRepoId,
					Name:            job.Spec.Repo,
					FullName:        fmt.Sprintf("%s/%s", job.Spec.Owner, job.Spec.Repo),
					Private:         false,
					StargazersCount: 0,
					UpdatedAt:       ji.UpdatedAt,
				},
				AnalysisStatus:    status.ToExternalString(),
				ResultCount:       resultCount,
				ArtifactSizeBytes: int(artifactSize),
			},
		)
	}

	jobStatus, err := c.v.State.GetStatus(js)
	if err != nil {
		slog.Error("Error getting job status", "error", err.Error())
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	status := common.StatusResponse{
		SessionId:            js.SessionID,
		ControllerRepo:       common.ControllerRepo{},
		Actor:                common.Actor{},
		QueryLanguage:        ji.QueryLanguage,
		QueryPackURL:         "", // FIXME
		CreatedAt:            ji.CreatedAt,
		UpdatedAt:            ji.UpdatedAt,
		ActionsWorkflowRunID: -1, // FIXME
		Status:               jobStatus.ToExternalString(),
		ScannedRepositories:  scannedRepos,
		SkippedRepositories:  ji.SkippedRepositories,
	}

	// Encode the response as JSON
	submitStatus, err := json.Marshal(status)
	if err != nil {
		slog.Error("Error encoding response as JSON:",
			"error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send analysisReposJSON via ResponseWriter
	w.Header().Set("Content-Type", "application/json")
	w.Write(submitStatus)
}

func (c *CommanderSingle) RootHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Request on /")
}

func (c *CommanderSingle) MRVAStatusCommon(w http.ResponseWriter, r *http.Request, owner, repo string, variantAnalysisID string) {
	slog.Info("MRVA status request for ",
		"owner", owner,
		"repo", repo,
		"codeql_variant_analysis_id", variantAnalysisID)

	sessionId, err := strconv.ParseInt(variantAnalysisID, 10, 32)
	if err != nil {
		slog.Error("Variant analysis ID is not integer", "id",
			variantAnalysisID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobs, err := c.v.State.GetJobList(int(sessionId))
	if err != nil {
		msg := "No jobs found for given session id"
		slog.Error(msg, "id", variantAnalysisID)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	if len(jobs) == 0 {
		c.submitEmptyStatusResponse(w, int(sessionId))
		return
	}

	// The status reports one status for all jobs belonging to an id.
	// So we simply report the status of a job as the status of all.
	// TODO: verify this behaviour
	job := jobs[0]

	jobInfo, err := c.v.State.GetJobInfo(job.Spec)
	if err != nil {
		msg := "No job info found for given session id"
		slog.Error(msg, "id", variantAnalysisID)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	c.submitStatusResponse(w, job.Spec, jobInfo)

}

func (c *CommanderSingle) MRVAStatusID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("MRVA status request (MRVAStatusID)")
	// Mapping to unused/unused and passing variant analysis id
	c.MRVAStatusCommon(w, r, "unused", "unused", vars["codeql_variant_analysis_id"])
}

func (c *CommanderSingle) MRVAStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("MRVA status request (MRVAStatus)")
	// Mapping to owner/repo and passing variant analysis id
	c.MRVAStatusCommon(w, r, vars["owner"], vars["repo"], vars["codeql_variant_analysis_id"])
}

// Download artifacts
func (c *CommanderSingle) MRVADownloadArtifactCommon(w http.ResponseWriter, r *http.Request, jobRepoId int, jobSpec common.JobSpec) {
	slog.Debug("MRVA artifact download",
		"codeql_variant_analysis_id", jobSpec.SessionID,
		"repo_owner", jobSpec.NameWithOwner.Owner,
		"repo_name", jobSpec.NameWithOwner.Repo,
	)

	c.sendArtifactDownloadResponse(w, jobRepoId, jobSpec)
}

func (c *CommanderSingle) MRVADownloadArtifactID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Debug("MRVA artifact download", "id", vars["codeql_variant_analysis_id"], "repo_id", vars["repository_id"])

	sessionId, err := strconv.ParseInt(vars["codeql_variant_analysis_id"], 10, 32)
	if err != nil {
		slog.Error("Variant analysis ID is not an integer", "id", vars["codeql_variant_analysis_id"])
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// this must match the repo ID returned by the status request
	repoId, err := strconv.ParseInt(vars["repository_id"], 10, 32)
	if err != nil {
		slog.Error("Repository ID is not an integer", "id", vars["repository_id"])
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobSpec, err := c.v.State.GetJobSpecByRepoId(int(sessionId), int(repoId))
	if err != nil {
		slog.Error("Failed to get job spec by repo ID", "error", err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	c.MRVADownloadArtifactCommon(w, r, int(repoId), jobSpec)
}

func (c *CommanderSingle) MRVADownloadQLDB(w http.ResponseWriter, r *http.Request) {
	// The repositories are uploaded without language and can be downloaded
	// without it.  We ignore the language parameter passed in the request:
	// vars["repo_language"]

	// Other artifact downloads, like sendArtifactDownloadResponse, depend on
	// a jobspec (integer job id).  This request has none, and needs none.

	// An original upload example is
	//		tdlib$telegram-bot-apictsj8529d9.zip to bucket qldb.

	// This is a direct data request -- don't reply with a download url.

	vars := mux.Vars(r)
	dbl := common.NameWithOwner{
		Owner: vars["repo_owner"],
		Repo:  vars["repo_name"],
	}

	slog.Debug("Returning codeql database using database location",
		"dbl", dbl,
	)

	dbContent, err := c.v.CodeQLDBStore.GetDatabase(dbl)
	if err != nil {
		slog.Error("Failed to retrieve ql database",
			"error", err,
			"dbl", dbl,
		)
		http.Error(w, "Failed to retrieve ql database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(dbContent)

}

func (c *CommanderSingle) MRVADownloadArtifact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	sessionId, err := strconv.ParseInt(vars["codeql_variant_analysis_id"], 10, 32)
	if err != nil {
		slog.Error("Variant analysis ID is not an integer", "id", vars["codeql_variant_analysis_id"])
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobSpec := common.JobSpec{
		SessionID: int(sessionId),
		NameWithOwner: common.NameWithOwner{
			Owner: vars["repo_owner"],
			Repo:  vars["repo_name"],
		},
	}
	// TODO: THIS IS BROKEN UNLESS REPO ID IS IGNORED
	c.MRVADownloadArtifactCommon(w, r, -1, jobSpec)
}

func (c *CommanderSingle) sendArtifactDownloadResponse(w http.ResponseWriter, jobRepoId int, jobSpec common.JobSpec) {
	var response common.DownloadResponse

	slog.Debug("Forming download response", "job", jobSpec)

	jobStatus, err := c.v.State.GetStatus(jobSpec)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if jobStatus == common.StatusSuccess {
		jobResult, err := c.v.State.GetResult(jobSpec)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		jobResultData, err := c.v.Artifacts.GetResult(jobResult.ResultLocation)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Generate the artifact URL
		encodedJobSpec, err := common.EncodeJobSpec(jobSpec)
		if err != nil {
			http.Error(w, "Failed to encode job spec", http.StatusInternalServerError)
			return
		}

		// TODO: document/make less hacky
		host := os.Getenv("SERVER_HOST")
		if host == "" {
			host = "localhost"
		}

		port := os.Getenv("SERVER_PORT")
		if port == "" {
			port = "8080"
		}

		artifactURL := fmt.Sprintf("http://%s:%s/download/%s", host, port, encodedJobSpec)

		response = common.DownloadResponse{
			Repository: common.DownloadRepo{
				// TODO: fix jobRepoID coming from the NWO path. The MRVA extension uses repo ID.
				ID:       jobRepoId,
				Name:     jobSpec.Repo,
				FullName: fmt.Sprintf("%s/%s", jobSpec.Owner, jobSpec.Repo),
			},
			AnalysisStatus:       jobStatus.ToExternalString(),
			ResultCount:          jobResult.ResultCount,
			ArtifactSizeBytes:    len(jobResultData),
			DatabaseCommitSha:    jobResult.DatabaseSHA,
			SourceLocationPrefix: jobResult.SourceLocationPrefix,
			ArtifactURL:          artifactURL,
		}
	} else {
		// not successful status
		response = common.DownloadResponse{
			Repository: common.DownloadRepo{
				// TODO: fix jobRepoID coming from the NWO path. The MRVA extension uses repo ID.
				ID:       jobRepoId,
				Name:     jobSpec.Repo,
				FullName: fmt.Sprintf("%s/%s", jobSpec.Owner, jobSpec.Repo),
			},
			AnalysisStatus:       jobStatus.ToExternalString(),
			ResultCount:          0,
			ArtifactSizeBytes:    0,
			DatabaseCommitSha:    "",
			SourceLocationPrefix: "",
			ArtifactURL:          "",
		}
	}

	// Encode the response as JSON
	responseJson, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error encoding response as JSON:",
			"error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send analysisReposJSON via ResponseWriter
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func (c *CommanderSingle) MRVADownloadServe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	encodedJobSpec := vars["encoded_job_spec"]

	jobSpec, err := common.DecodeJobSpec(encodedJobSpec)
	if err != nil {
		http.Error(w, "Invalid job spec", http.StatusBadRequest)
		return
	}

	slog.Info("Result download request", "job_spec", jobSpec)

	result, err := c.v.State.GetResult(jobSpec)
	if err != nil {
		slog.Error("Failed to get result", "error", err)
		http.Error(w, "Failed to get result", http.StatusInternalServerError)
		return
	}

	slog.Debug("Result location", "location", result.ResultLocation)

	data, err := c.v.Artifacts.GetResult(result.ResultLocation)
	if err != nil {
		slog.Error("Failed to retrieve artifact", "error", err)
		http.Error(w, "Failed to retrieve artifact", http.StatusInternalServerError)
		return
	}

	// Send the file as a response
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

func (c *CommanderSingle) MRVARequestCommon(w http.ResponseWriter, r *http.Request) {
	sessionId := c.v.State.NextID()
	slog.Info("New MRVA Request", "id", fmt.Sprint(sessionId))
	queryLanguage, repoNWOs, queryPackLocation, err := c.collectRequestInfoAndSaveQueryPack(w, r, sessionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Debug("Processed request info", "location", queryPackLocation, "language", queryLanguage)

	notFoundRepos, analysisRepos := c.v.CodeQLDBStore.FindAvailableDBs(repoNWOs)

	if len(analysisRepos) == 0 {
		slog.Warn("No repositories found for analysis")
	}

	// XX: session_is is separate from the query pack ref.  Value may be equal.
	// QueryPackURL is returned to the client, separately from the ID.
	// The values may be equal here, but this is irrelevant
	c.startAnalyses(analysisRepos, queryPackLocation, sessionId, queryLanguage)

	sessionInfo := SessionInfo{
		ID: sessionId,

		QueryPack: strconv.Itoa(sessionId), // TODO
		Language:  queryLanguage,

		AccessMismatchRepos: nil, /* FIXME */
		NotFoundRepos:       notFoundRepos,
		NoCodeqlDBRepos:     nil, /* FIXME */

	}

	slog.Debug("Forming and sending response for submitted analysis job", "id", sessionInfo.ID)
	submitResponseJson, err := c.buildSessionInfoResponseJson(sessionInfo)
	if err != nil {
		slog.Error("Error forming submit response", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(submitResponseJson)
}

func (c *CommanderSingle) MRVARequestID(w http.ResponseWriter, r *http.Request) {
	slog.Debug("MRVARequestID")
	c.MRVARequestCommon(w, r)
}

func (c *CommanderSingle) MRVARequest(w http.ResponseWriter, r *http.Request) {
	slog.Debug("MRVARequest")
	c.MRVARequestCommon(w, r)
}

func nwoToNwoStringArray(nwo []common.NameWithOwner) ([]string, int) {
	repos := []string{}
	count := len(nwo)
	for _, repo := range nwo {
		repos = append(repos, fmt.Sprintf("%s/%s", repo.Owner, repo.Repo))
	}
	return repos, count
}

func nwoToDummyRepositoryArray(nwo []common.NameWithOwner) ([]common.Repository, int) {
	repos := []common.Repository{}
	for _, repo := range nwo {
		repos = append(repos, common.Repository{
			ID:              -1,
			Name:            repo.Repo,
			FullName:        fmt.Sprintf("%s/%s", repo.Owner, repo.Repo),
			Private:         false,
			StargazersCount: 0,
			UpdatedAt:       time.Now().Format(time.RFC3339),
		})
	}
	count := len(nwo)

	return repos, count
}

// ConsumeResults moves results from 'queue' to server 'state'
func (c *CommanderSingle) ConsumeResults() {
	slog.Info("Started server results consumer.")
	for {
		r := <-c.v.Queue.Results()
		slog.Debug("Result consumed:", "r", r, "status", r.Status.ToExternalString())
		c.v.State.SetResult(r.Spec, r)
		c.v.State.SetStatus(r.Spec, r.Status)
	}
}

func (c *CommanderSingle) buildSessionInfoResponseJson(si SessionInfo) ([]byte, error) {
	// Construct the response bottom-up
	var controllerRepo common.ControllerRepo
	var actor common.Actor

	repoNames, count := nwoToNwoStringArray(si.NotFoundRepos)
	notFoundRepos := common.NotFoundRepos{RepositoryCount: count, RepositoryFullNames: repoNames}

	repos, _ := nwoToDummyRepositoryArray(si.AccessMismatchRepos)
	accessMismatchRepos := common.AccessMismatchRepos{RepositoryCount: count, Repositories: repos}

	repos, _ = nwoToDummyRepositoryArray(si.NoCodeqlDBRepos)
	noCodeQLDBRepos := common.NoCodeqlDBRepos{RepositoryCount: count, Repositories: repos}

	// TODO fill these with real values?
	repos, _ = nwoToDummyRepositoryArray(si.NoCodeqlDBRepos)
	overlimitRepos := common.OverLimitRepos{RepositoryCount: count, Repositories: repos}

	skippedRepositories := common.SkippedRepositories{
		AccessMismatchRepos: accessMismatchRepos,
		NotFoundRepos:       notFoundRepos,
		NoCodeqlDBRepos:     noCodeQLDBRepos,
		OverLimitRepos:      overlimitRepos}

	response := common.SubmitResponse{
		Actor:               actor,
		ControllerRepo:      controllerRepo,
		ID:                  si.ID,
		QueryLanguage:       string(si.Language),
		QueryPackURL:        si.QueryPack,
		CreatedAt:           time.Now().Format(time.RFC3339),
		UpdatedAt:           time.Now().Format(time.RFC3339),
		Status:              "in_progress",
		SkippedRepositories: skippedRepositories,
	}

	// Store data needed later
	joblist, err := c.v.State.GetJobList(si.ID)
	if err != nil {
		slog.Error("Error getting job list", "error", err.Error())
		return nil, err
	}

	for _, job := range joblist {
		c.v.State.SetJobInfo(common.JobSpec{
			SessionID:     si.ID,
			NameWithOwner: job.Spec.NameWithOwner,
		}, common.JobInfo{
			QueryLanguage:       string(si.Language),
			CreatedAt:           response.CreatedAt,
			UpdatedAt:           response.UpdatedAt,
			SkippedRepositories: skippedRepositories,
		},
		)
	}

	// Encode the response as JSON
	responseJson, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error encoding response as JSON", "err", err)
		return nil, err
	}
	return responseJson, nil

}

func (c *CommanderSingle) collectRequestInfoAndSaveQueryPack(w http.ResponseWriter, r *http.Request, sessionId int) (queue.QueryLanguage, []common.NameWithOwner, artifactstore.ArtifactLocation, error) {
	slog.Debug("Collecting session info")

	if r.Body == nil {
		err := errors.New("missing request body")
		slog.Error("Error reading MRVA submission body", "error", err)
		http.Error(w, err.Error(), http.StatusNoContent)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		var w http.ResponseWriter
		slog.Error("Error reading MRVA submission body", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}

	msg, err := tryParseSubmitMsg(buf)
	if err != nil {
		slog.Error("Unknown MRVA submission body format", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}

	// 1. Save the query pack and keep the location
	if !utils.IsBase64Gzip([]byte(msg.QueryPack)) {
		slog.Error("MRVA submission body querypack has invalid format")
		err := errors.New("MRVA submission body querypack has invalid format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}

	queryPackLocation, err := c.decodeAndSaveBase64QueryPack(msg.QueryPack, sessionId)
	if err != nil {
		slog.Error("Error processing query pack archive", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}

	// 2. Save the language
	sessionLanguage := queue.QueryLanguage(msg.Language)

	// 3. Save the repositories
	var sessionRepos []common.NameWithOwner

	for _, v := range msg.Repositories {
		t := strings.Split(v, "/")
		if len(t) != 2 {
			err := "Invalid owner / repository entry"
			slog.Error(err, "entry", t)
			http.Error(w, err, http.StatusBadRequest)
		}
		sessionRepos = append(sessionRepos,
			common.NameWithOwner{Owner: t[0], Repo: t[1]})
	}

	return sessionLanguage, sessionRepos, queryPackLocation, nil
}

// Try to extract a SubmitMsg from a json-encoded buffer
func tryParseSubmitMsg(buf []byte) (common.SubmitMsg, error) {
	buf1 := make([]byte, len(buf))
	copy(buf1, buf)
	dec := json.NewDecoder(bytes.NewReader(buf1))
	dec.DisallowUnknownFields()
	var m common.SubmitMsg
	err := dec.Decode(&m)
	return m, err
}

func (c *CommanderSingle) decodeAndSaveBase64QueryPack(qp string, sessionId int) (artifactstore.ArtifactLocation, error) {
	// These are decoded manually via
	//    base64 -d < foo1 | gunzip | tar t | head -20
	// base64 decode the body
	slog.Debug("Extracting query pack")

	tgz, err := base64.StdEncoding.DecodeString(qp)
	if err != nil {
		slog.Error("Failed to decode query pack body", "err", err)
		return artifactstore.ArtifactLocation{}, err
	}

	// XX: afl use
	artifactLocation, err := c.v.Artifacts.SaveQueryPack(sessionId, tgz)
	if err != nil {
		slog.Error("Failed to save query pack", "err", err)
		return artifactstore.ArtifactLocation{}, err
	}

	return artifactLocation, err
}
