package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/qldbstore"

	"github.com/gorilla/mux"
)

func (c *CommanderContainer) startAnalyses(
	// X1: check
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

func (c *CommanderContainer) StatusResponse(w http.ResponseWriter, js common.JobSpec, ji common.JobInfo, vaid int) {
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

func (c *CommanderContainer) RootHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Request on /")
}

func (c *CommanderContainer) MRVAStatus(w http.ResponseWriter, r *http.Request) {
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
func (c *CommanderContainer) MRVADownloadArtifact(w http.ResponseWriter, r *http.Request) {
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

func (c *CommanderContainer) DownloadResponse(w http.ResponseWriter, js common.JobSpec, jobID int) {
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

func (c *CommanderContainer) MRVADownloadServe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("File download request", "local_path", vars["local_path"])

	FileDownload(w, vars["local_path"])
}

func (c *CommanderContainer) MRVARequestID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva run using repository ID", "id", vars["repository_id"])
}

func (c *CommanderContainer) MRVARequest(w http.ResponseWriter, r *http.Request) {
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

func (c *CommanderContainer) submitResponse(si SessionInfo) ([]byte, error) {
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

func (c *CommanderContainer) collectRequestInfo(w http.ResponseWriter, r *http.Request, sessionId int) (string, []common.NameWithOwner, artifactstore.ArtifactLocation, error) {
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

func (c *CommanderContainer) processQueryPackArchive(qp string, sessionID int) (artifactstore.ArtifactLocation, error) {
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
