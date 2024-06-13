package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/storage"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func (c *CommanderContainer) Setup(st *Visibles) {
	c.st = st
	setupEndpoints(c)
}

func (c *CommanderContainer) StatusResponse(w http.ResponseWriter, js common.JobSpec, ji common.JobInfo, vaid int) {
	slog.Debug("Submitting status response", "session", vaid)

	all_scanned := []common.ScannedRepo{}
	// XX:
	jobs := storage.GetJobList(js.JobID)
	for _, job := range jobs {
		astat := storage.GetStatus(js.JobID, job.ORepo).ToExternalString()
		all_scanned = append(all_scanned,
			common.ScannedRepo{
				Repository: common.Repository{
					ID:              0,
					Name:            job.ORepo.Repo,
					FullName:        fmt.Sprintf("%s/%s", job.ORepo.Owner, job.ORepo.Repo),
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

	// XX:
	astat := storage.GetStatus(js.JobID, js.OwnerRepo).ToExternalString()

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

func (c *CommanderContainer) MirvaStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("mrva status request for ",
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
	// XX:
	spec := storage.GetJobList(id)
	if spec == nil {
		msg := "No jobs found for given job id"
		slog.Error(msg, "id", vars["codeql_variant_analysis_id"])
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}

	job := spec[0]

	js := common.JobSpec{
		JobID:     job.QueryPackId,
		OwnerRepo: job.ORepo,
	}

	// XX:
	ji := storage.GetJobInfo(js)

	c.StatusResponse(w, js, ji, id)
}

// Download artifacts
func (c *CommanderContainer) MirvaDownloadArtifact(w http.ResponseWriter, r *http.Request) {
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
		OwnerRepo: common.OwnerRepo{
			Owner: vars["repo_owner"],
			Repo:  vars["repo_name"],
		},
	}
	c.DownloadResponse(w, js, vaid)
}

func (c *CommanderContainer) DownloadResponse(w http.ResponseWriter, js common.JobSpec, vaid int) {
	slog.Debug("Forming download response", "session", vaid, "job", js)

	// XX:
	astat := storage.GetStatus(vaid, js.OwnerRepo)

	var dlr common.DownloadResponse
	if astat == common.StatusSuccess {

		// XX:
		au, err := storage.ArtifactURL(js, vaid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dlr = common.DownloadResponse{
			Repository: common.DownloadRepo{
				Name:     js.Repo,
				FullName: fmt.Sprintf("%s/%s", js.Owner, js.Repo),
			},
			AnalysisStatus:       astat.ToExternalString(),
			ResultCount:          123, // FIXME
			ArtifactSizeBytes:    123, // FIXME
			DatabaseCommitSha:    "do-we-use-dcs-p",
			SourceLocationPrefix: "do-we-use-slp-p",
			ArtifactURL:          au,
		}
	} else {
		dlr = common.DownloadResponse{
			Repository: common.DownloadRepo{
				Name:     js.Repo,
				FullName: fmt.Sprintf("%s/%s", js.Owner, js.Repo),
			},
			AnalysisStatus:       astat.ToExternalString(),
			ResultCount:          0,
			ArtifactSizeBytes:    0,
			DatabaseCommitSha:    "",
			SourceLocationPrefix: "/not/relevant/here",
			ArtifactURL:          "",
		}
	}

	// Encode the response as JSON
	jdlr, err := json.Marshal(dlr)
	if err != nil {
		slog.Error("Error encoding response as JSON:",
			"error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send analysisReposJSON via ResponseWriter
	w.Header().Set("Content-Type", "application/json")
	w.Write(jdlr)

}

func (c *CommanderContainer) MirvaDownloadServe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("File download request", "local_path", vars["local_path"])

	FileDownload(w, vars["local_path"])
}

func (c *CommanderContainer) MirvaRequestID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva using repository_id=%v\n", vars["repository_id"])
}

func (c *CommanderContainer) MirvaRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva run ", "owner", vars["owner"], "repo", vars["repo"])

	session_id := c.st.ServerStore.NextID()
	session_owner := vars["owner"]
	session_controller_repo := vars["repo"]
	slog.Info("new run", "id: ", fmt.Sprint(session_id), session_owner, session_controller_repo)
	session_language, session_repositories, session_tgz_ref, err := c.collectRequestInfo(w, r, session_id)
	if err != nil {
		return
	}

	not_found_repos, analysisRepos := c.st.ServerStore.FindAvailableDBs(session_repositories)

	c.st.Queue.StartAnalyses(analysisRepos, session_id, session_language)

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
	submit_response, err := submit_response(si)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(submit_response)
}

func (c *CommanderContainer) collectRequestInfo(w http.ResponseWriter, r *http.Request, sessionId int) (string, []common.OwnerRepo, string, error) {
	slog.Debug("Collecting session info")

	if r.Body == nil {
		err := errors.New("missing request body")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNoContent)
		return "", []common.OwnerRepo{}, "", err
	}
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		var w http.ResponseWriter
		slog.Error("Error reading MRVA submission body", "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.OwnerRepo{}, "", err
	}
	msg, err := TrySubmitMsg(buf)
	if err != nil {
		// Unknown message
		slog.Error("Unknown MRVA submission body format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.OwnerRepo{}, "", err
	}
	// Decompose the SubmitMsg and keep information

	// Save the query pack and keep the location
	if !isBase64Gzip([]byte(msg.QueryPack)) {
		slog.Error("MRVA submission body querypack has invalid format")
		err := errors.New("MRVA submission body querypack has invalid format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.OwnerRepo{}, "", err
	}
	session_tgz_ref, err := c.extract_tgz(msg.QueryPack, sessionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.OwnerRepo{}, "", err
	}

	// 2. Save the language
	session_language := msg.Language

	// 3. Save the repositories
	var session_repositories []common.OwnerRepo

	for _, v := range msg.Repositories {
		t := strings.Split(v, "/")
		if len(t) != 2 {
			err := "Invalid owner / repository entry"
			slog.Error(err, "entry", t)
			http.Error(w, err, http.StatusBadRequest)
		}
		session_repositories = append(session_repositories,
			common.OwnerRepo{Owner: t[0], Repo: t[1]})
	}
	return session_language, session_repositories, session_tgz_ref, nil
}

func (c *CommanderContainer) extract_tgz(qp string, sessionID int) (string, error) {
	// These are decoded manually via
	//    base64 -d < foo1 | gunzip | tar t | head -20
	// base64 decode the body
	slog.Debug("Extracting query pack")

	tgz, err := base64.StdEncoding.DecodeString(qp)
	if err != nil {
		slog.Error("querypack body decoding error:", err)
		return "", err
	}

	session_query_pack_tgz_filepath, err := c.st.ServerStore.SaveQueryPack(tgz, sessionID)
	if err != nil {
		return "", err
	}

	return session_query_pack_tgz_filepath, err
}
