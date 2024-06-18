package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/qldbstore"
	"mrvacommander/pkg/state"

	"github.com/gorilla/mux"
)

func (c *CommanderSingle) startAnalyses(
	analysisRepos *map[common.NameWithOwner]qldbstore.CodeQLDatabaseLocation,
	jobID int,
	queryLanguage string) {
	slog.Debug("Queueing analysis jobs")

	for nwo := range *analysisRepos {
		info := common.AnalyzeJob{
			QueryPackId:   jobID,
			QueryLanguage: queryLanguage,
			NWO:           nwo,
		}
		c.v.Queue.Jobs() <- info
		c.v.State.SetStatus(jobID, nwo, common.StatusQueued)
		c.v.State.AddJob(jobID, info)
	}
}

func setupEndpoints(c CommanderAPI) {
	r := mux.NewRouter()

	// API endpoints that mirror those used in the GitHub API
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses", c.MRVARequest)
	// Example: /repos/hohn/mrva-controller/code-scanning/codeql/variant-analyses

	// Endpoint using repository ID
	r.HandleFunc("/{repository_id}/code-scanning/codeql/variant-analyses", c.MRVARequestID)

	// Root handler
	r.HandleFunc("/", c.RootHandler)

	// Standalone status request
	// This is also the first request made when downloading; the difference is in the client-side handling.
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}", c.MRVAStatus)

	// Endpoint for downloading artifacts
	r.HandleFunc("/repos/{controller_owner}/{controller_repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}/repos/{repo_owner}/{repo_name}", c.MRVADownloadArtifact)

	// Not implemented:
	// r.HandleFunc("/codeql-query-console/codeql-variant-analysis-repo-tasks/{codeql_variant_analysis_id}/{repo_id}/{owner_id}/{controller_repo_id}", MRVADownLoad3)
	// r.HandleFunc("/github-codeql-query-console-prod/codeql-variant-analysis-repo-tasks/{codeql_variant_analysis_id}/{repo_id}", MRVADownLoad4)

	// Support API endpoint
	r.HandleFunc("/download-server/{local_path:.*}", c.MRVADownloadServe)

	// Bind to a port and pass our router in
	// TODO: Make this a configuration entry
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		slog.Error("Error starting server:", "error", err)
		os.Exit(1)
	}
}

func (c *CommanderSingle) StatusResponse(w http.ResponseWriter, js common.JobSpec, ji common.JobInfo, vaid int) {
	slog.Debug("Submitting status response", "session", vaid)

	all_scanned := []common.ScannedRepo{}
	jobs := c.v.State.GetJobList(js.JobID)
	for _, job := range jobs {
		astat := c.v.State.GetStatus(js.JobID, job.NWO).ToExternalString()
		all_scanned = append(all_scanned,
			common.ScannedRepo{
				Repository: common.Repository{
					ID:              0,
					Name:            job.NWO.Repo,
					FullName:        fmt.Sprintf("%s/%s", job.NWO.Owner, job.NWO.Repo),
					Private:         false,
					StargazersCount: 0,
					UpdatedAt:       ji.UpdatedAt,
				},
				AnalysisStatus:    astat,
				ResultCount:       123, // FIXME  123 is a lie so the client downloads
				ArtifactSizeBytes: 123, // FIXME
			},
		)
	}

	astat := c.v.State.GetStatus(js.JobID, js.NameWithOwner).ToExternalString()

	status := common.StatusResponse{
		SessionId:            js.JobID,
		ControllerRepo:       common.ControllerRepo{},
		Actor:                common.Actor{},
		QueryLanguage:        ji.QueryLanguage,
		QueryPackURL:         "", // FIXME
		CreatedAt:            ji.CreatedAt,
		UpdatedAt:            ji.UpdatedAt,
		ActionsWorkflowRunID: 0, // FIXME
		Status:               astat,
		ScannedRepositories:  all_scanned,
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

func (c *CommanderSingle) MRVAStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("MRVA status request for ",
		"owner", vars["owner"],
		"repo", vars["repo"],
		"codeql_variant_analysis_id", vars["codeql_variant_analysis_id"])
	id, err := strconv.Atoi(vars["codeql_variant_analysis_id"])
	if err != nil {
		slog.Error("Variant analysis is is not integer", "id",
			vars["codeql_variant_analysis_id"])
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// The status reports one status for all jobs belonging to an id.
	// So we simply report the status of a job as the status of all.
	spec := c.v.State.GetJobList(id)
	if spec == nil {
		msg := "No jobs found for given job id"
		slog.Error(msg, "id", vars["codeql_variant_analysis_id"])
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}

	job := spec[0]

	js := common.JobSpec{
		JobID:         job.QueryPackId,
		NameWithOwner: job.NWO,
	}

	ji := c.v.State.GetJobInfo(js)

	c.StatusResponse(w, js, ji, id)
}

// Download artifacts
func (c *CommanderSingle) MRVADownloadArtifact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("MRVA artifact download",
		"controller_owner", vars["controller_owner"],
		"controller_repo", vars["controller_repo"],
		"codeql_variant_analysis_id", vars["codeql_variant_analysis_id"],
		"repo_owner", vars["repo_owner"],
		"repo_name", vars["repo_name"],
	)
	vaid, err := strconv.Atoi(vars["codeql_variant_analysis_id"])
	if err != nil {
		slog.Error("Variant analysis is is not integer", "id",
			vars["codeql_variant_analysis_id"])
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js := common.JobSpec{
		JobID: vaid,
		NameWithOwner: common.NameWithOwner{
			Owner: vars["repo_owner"],
			Repo:  vars["repo_name"],
		},
	}
	c.DownloadResponse(w, js, vaid)
}

func (c *CommanderSingle) DownloadResponse(w http.ResponseWriter, js common.JobSpec, jobID int) {
	var response common.DownloadResponse

	slog.Debug("Forming download response", "id", jobID, "job", js)

	jobStatus := c.v.State.GetStatus(jobID, js.NameWithOwner)

	if jobStatus == common.StatusSuccess {
		jobResult := c.v.State.GetResult(js)
		// TODO: return this as a URL @hohn
		jobResultData, err := c.v.Artifacts.GetResult(jobResult.ResultLocation)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response = common.DownloadResponse{
			Repository: common.DownloadRepo{
				Name:     js.Repo,
				FullName: fmt.Sprintf("%s/%s", js.Owner, js.Repo),
			},
			AnalysisStatus:       jobStatus.ToExternalString(),
			ResultCount:          jobResult.ResultCount,
			ArtifactSizeBytes:    len(jobResultData),
			DatabaseCommitSha:    "do-we-use-dcs-p",
			SourceLocationPrefix: "do-we-use-slp-p",
			ArtifactURL:          "TODO", // @hohn
		}
	} else {
		response = common.DownloadResponse{
			Repository: common.DownloadRepo{
				Name:     js.Repo,
				FullName: fmt.Sprintf("%s/%s", js.Owner, js.Repo),
			},
			AnalysisStatus:       jobStatus.ToExternalString(),
			ResultCount:          0,
			ArtifactSizeBytes:    0,
			DatabaseCommitSha:    "",
			SourceLocationPrefix: "/not/relevant/here",
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
	slog.Info("File download request", "local_path", vars["local_path"])

	FileDownload(w, vars["local_path"])
}

func FileDownload(w http.ResponseWriter, path string) {
	slog.Debug("Sending zip file with .sarif/.bqrs", "path", path)

	fpath, res, err := state.ResultAsFile(path)
	if err != nil {
		http.Error(w, "Failed to read results", http.StatusInternalServerError)
		return
	}
	// Set headers
	fname := filepath.Base(fpath)
	w.Header().Set("Content-Disposition", "attachment; filename="+fname)
	w.Header().Set("Content-Type", "application/octet-stream")

	// Copy the file contents to the response writer
	rdr := bytes.NewReader(res)
	_, err = io.Copy(w, rdr)
	if err != nil {
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
		return
	}

	slog.Debug("Uploaded file", "path", fpath)
}

func (c *CommanderSingle) MRVARequestID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva run using repository ID", "id", vars["repository_id"])
}

func (c *CommanderSingle) MRVARequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva run", "owner", vars["owner"], "repo", vars["repo"])

	session_id := c.v.State.NextID()
	session_owner := vars["owner"]
	session_controller_repo := vars["repo"]
	slog.Info("new run", "id: ", fmt.Sprint(session_id), session_owner, session_controller_repo)
	session_language, session_repositories, session_tgz_ref, err := c.collectRequestInfo(w, r, session_id)
	if err != nil {
		return
	}

	not_found_repos, analysisRepos := c.v.CodeQLDBStore.FindAvailableDBs(session_repositories)

	c.startAnalyses(analysisRepos, session_id, session_language)

	si := SessionInfo{
		ID:             session_id,
		Owner:          session_owner,
		ControllerRepo: session_controller_repo,

		QueryPack:    session_tgz_ref,
		Language:     session_language,
		Repositories: session_repositories,

		AccessMismatchRepos: nil, /* FIXME */
		NotFoundRepos:       not_found_repos,
		NoCodeqlDBRepos:     nil, /* FIXME */
		OverLimitRepos:      nil, /* FIXME */

		AnalysisRepos: analysisRepos,
	}

	slog.Debug("Forming and sending response for submitted analysis job", "id", si.ID)
	submit_response, err := c.submitResponse(si)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(submit_response)
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
			ID:              0,
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

func (c *CommanderSingle) submitResponse(si SessionInfo) ([]byte, error) {
	// Construct the response bottom-up
	var m_cr common.ControllerRepo
	var m_ac common.Actor

	repos, count := nwoToNwoStringArray(si.NotFoundRepos)
	r_nfr := common.NotFoundRepos{RepositoryCount: count, RepositoryFullNames: repos}

	ra, rac := nwoToDummyRepositoryArray(si.AccessMismatchRepos)
	r_amr := common.AccessMismatchRepos{RepositoryCount: rac, Repositories: ra}

	ra, rac = nwoToDummyRepositoryArray(si.NoCodeqlDBRepos)
	r_ncd := common.NoCodeqlDBRepos{RepositoryCount: rac, Repositories: ra}

	// TODO fill these with real values?
	ra, rac = nwoToDummyRepositoryArray(si.NoCodeqlDBRepos)
	r_olr := common.OverLimitRepos{RepositoryCount: rac, Repositories: ra}

	m_skip := common.SkippedRepositories{
		AccessMismatchRepos: r_amr,
		NotFoundRepos:       r_nfr,
		NoCodeqlDBRepos:     r_ncd,
		OverLimitRepos:      r_olr}

	m_sr := common.SubmitResponse{
		Actor:          m_ac,
		ControllerRepo: m_cr,
		ID:             si.ID,
		QueryLanguage:  si.Language,
		// TODO: broken, need proper URL using si.data
		QueryPackURL:        "broken-for-now",
		CreatedAt:           time.Now().Format(time.RFC3339),
		UpdatedAt:           time.Now().Format(time.RFC3339),
		Status:              "in_progress",
		SkippedRepositories: m_skip,
	}

	// Store data needed later
	// joblist := state.GetJobList(si.ID)
	// (si.JobID)?
	joblist := c.v.State.GetJobList(si.ID)

	for _, job := range joblist {
		c.v.State.SetJobInfo(common.JobSpec{
			JobID:         si.ID,
			NameWithOwner: job.NWO,
		}, common.JobInfo{
			QueryLanguage:       si.Language,
			CreatedAt:           m_sr.CreatedAt,
			UpdatedAt:           m_sr.UpdatedAt,
			SkippedRepositories: m_skip,
		},
		)
	}

	// Encode the response as JSON
	submit_response, err := json.Marshal(m_sr)
	if err != nil {
		slog.Warn("Error encoding response as JSON:", err.Error())
		return nil, err
	}
	return submit_response, nil

}

func (c *CommanderSingle) collectRequestInfo(w http.ResponseWriter, r *http.Request, sessionId int) (string, []common.NameWithOwner, artifactstore.ArtifactLocation, error) {
	slog.Debug("Collecting session info")

	if r.Body == nil {
		err := errors.New("missing request body")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNoContent)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		var w http.ResponseWriter
		slog.Error("Error reading MRVA submission body", "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}
	msg, err := TrySubmitMsg(buf)
	if err != nil {
		// Unknown message
		slog.Error("Unknown MRVA submission body format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}
	// Decompose the SubmitMsg and keep information

	// Save the query pack and keep the location
	if !isBase64Gzip([]byte(msg.QueryPack)) {
		slog.Error("MRVA submission body querypack has invalid format")
		err := errors.New("MRVA submission body querypack has invalid format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}
	session_tgz_ref, err := c.processQueryPackArchive(msg.QueryPack, sessionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, artifactstore.ArtifactLocation{}, err
	}

	// 2. Save the language
	session_language := msg.Language

	// 3. Save the repositories
	var session_repositories []common.NameWithOwner

	for _, v := range msg.Repositories {
		t := strings.Split(v, "/")
		if len(t) != 2 {
			err := "Invalid owner / repository entry"
			slog.Error(err, "entry", t)
			http.Error(w, err, http.StatusBadRequest)
		}
		session_repositories = append(session_repositories,
			common.NameWithOwner{Owner: t[0], Repo: t[1]})
	}
	return session_language, session_repositories, session_tgz_ref, nil
}

// Try to extract a SubmitMsg from a json-encoded buffer
func TrySubmitMsg(buf []byte) (common.SubmitMsg, error) {
	buf1 := make([]byte, len(buf))
	copy(buf1, buf)
	dec := json.NewDecoder(bytes.NewReader(buf1))
	dec.DisallowUnknownFields()
	var m common.SubmitMsg
	err := dec.Decode(&m)
	return m, err
}

// Some important payloads can be listed via
// base64 -d < foo1 | gunzip | tar t|head -20
//
// This function checks the request body up to the `gunzip` part.
func isBase64Gzip(val []byte) bool {
	if len(val) >= 4 {
		// Extract header
		hdr := make([]byte, base64.StdEncoding.DecodedLen(4))
		_, err := base64.StdEncoding.Decode(hdr, []byte(val[0:4]))
		if err != nil {
			log.Println("WARNING: IsBase64Gzip decode error:", err)
			return false
		}
		// Check for gzip heading
		magic := []byte{0x1f, 0x8b}
		if bytes.Equal(hdr[0:2], magic) {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (c *CommanderSingle) processQueryPackArchive(qp string, sessionID int) (artifactstore.ArtifactLocation, error) {
	// These are decoded manually via
	//    base64 -d < foo1 | gunzip | tar t | head -20
	// base64 decode the body
	slog.Debug("Extracting query pack")

	tgz, err := base64.StdEncoding.DecodeString(qp)
	if err != nil {
		slog.Error("querypack body decoding error:", err)
		return artifactstore.ArtifactLocation{}, err
	}

	session_query_pack_tgz_filepath, err := c.v.Artifacts.SaveQueryPack(sessionID, tgz)
	if err != nil {
		return artifactstore.ArtifactLocation{}, err
	}

	return session_query_pack_tgz_filepath, err
}
