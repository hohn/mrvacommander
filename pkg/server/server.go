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
	"strconv"
	"strings"

	"mrvacommander/pkg/storage"

	"github.com/gorilla/mux"
	"github.com/hohn/ghes-mirva-server/analyze"
	"github.com/hohn/ghes-mirva-server/api"
	co "github.com/hohn/ghes-mirva-server/common"
	"github.com/hohn/ghes-mirva-server/store"
)

func (c *CommanderSingle) Run() {
}

func (c *CommanderSingle) Setup(st *State) {
	r := mux.NewRouter()
	c.st = st

	//
	// First are the API endpoints that mirror those used in the github API
	//
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses", c.MirvaRequest)
	// 			  /repos/hohn   /mirva-controller/code-scanning/codeql/variant-analyses
	// Or via
	r.HandleFunc("/{repository_id}/code-scanning/codeql/variant-analyses", c.MirvaRequestID)

	r.HandleFunc("/", c.RootHandler)

	// This is the standalone status request.
	// It's also the first request made when downloading; the difference is on the
	// client side's handling.
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}", c.MirvaStatus)

	r.HandleFunc("/repos/{controller_owner}/{controller_repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}/repos/{repo_owner}/{repo_name}", c.MirvaDownloadArtifact)

	// Not implemented:
	// r.HandleFunc("/codeql-query-console/codeql-variant-analysis-repo-tasks/{codeql_variant_analysis_id}/{repo_id}/{owner_id}/{controller_repo_id}", MirvaDownLoad3)
	// r.HandleFunc("/github-codeql-query-console-prod/codeql-variant-analysis-repo-tasks/{codeql_variant_analysis_id}/{repo_id}", MirvaDownLoad4)

	//
	// Now some support API endpoints
	//
	r.HandleFunc("/download-server/{local_path:.*}", c.MirvaDownloadServe)

	//
	// Bind to a port and pass our router in
	//
	log.Fatal(http.ListenAndServe(":8080", r))
}

func (c *CommanderSingle) StatusResponse(w http.ResponseWriter, js co.JobSpec, ji co.JobInfo, vaid int) {
	slog.Debug("Submitting status response", "session", vaid)

	all_scanned := []api.ScannedRepo{}
	jobs := store.GetJobList(js.ID)
	for _, job := range jobs {
		astat := store.GetStatus(js.ID, job.ORL).ToExternalString()
		all_scanned = append(all_scanned,
			api.ScannedRepo{
				Repository: api.Repository{
					ID:              0,
					Name:            job.ORL.Repo,
					FullName:        fmt.Sprintf("%s/%s", job.ORL.Owner, job.ORL.Repo),
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

	astat := store.GetStatus(js.ID, js.OwnerRepo).ToExternalString()

	status := api.StatusResponse{
		SessionId:            js.ID,
		ControllerRepo:       api.ControllerRepo{},
		Actor:                api.Actor{},
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

func (c *CommanderSingle) MirvaStatus(w http.ResponseWriter, r *http.Request) {
	// 	TODO Port this function from ghes-mirva-server
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
	spec := store.GetJobList(id)
	if spec == nil {
		msg := "No jobs found for given job id"
		slog.Error(msg, "id", vars["codeql_variant_analysis_id"])
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}

	job := spec[0]

	js := co.JobSpec{
		ID:        job.QueryPackId,
		OwnerRepo: job.ORL,
	}

	ji := store.GetJobInfo(js)

	c.StatusResponse(w, js, ji, id)
}

// Download artifacts
func (c *CommanderSingle) MirvaDownloadArtifact(w http.ResponseWriter, r *http.Request) {
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
	js := co.JobSpec{
		ID: vaid,
		OwnerRepo: co.OwnerRepo{
			Owner: vars["repo_owner"],
			Repo:  vars["repo_name"],
		},
	}
	c.DownloadResponse(w, js, vaid)
}

func (c *CommanderSingle) DownloadResponse(w http.ResponseWriter, js co.JobSpec, vaid int) {
	slog.Debug("Forming download response", "session", vaid, "job", js)

	astat := store.GetStatus(vaid, js.OwnerRepo)

	var dlr api.DownloadResponse
	if astat == co.StatusSuccess {

		au, err := storage.ArtifactURL(js, vaid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dlr = api.DownloadResponse{
			Repository: api.DownloadRepo{
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
		dlr = api.DownloadResponse{
			Repository: api.DownloadRepo{
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

func (c *CommanderSingle) MirvaDownloadServe(w http.ResponseWriter, r *http.Request) {
	// 	TODO Port this function from ghes-mirva-server
	vars := mux.Vars(r)
	slog.Info("File download request", "local_path", vars["local_path"])

	analyze.FileDownload(w, vars["local_path"])
}

func (c *CommanderSingle) MirvaRequestID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva using repository_id=%v\n", vars["repository_id"])
}

func (c *CommanderSingle) MirvaRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva run ", "owner", vars["owner"], "repo", vars["repo"])
	// session := new(MirvaSession)
	session_id := c.st.Storage.NextID()
	session_owner := vars["owner"]
	session_controller_repo := vars["repo"]
	slog.Info("new run", "id: ", fmt.Sprint(session_id), session_owner, session_controller_repo)

	session_language, session_repositories, session_tgz_ref, err := c.collectRequestInfo(w, r, session_id)

	if err != nil {
		return
	}

	not_found_repos, analysisRepos := c.st.Storage.FindAvailableDBs(session_repositories)

	// TODO into Queue
	// session_start_analyses()

	// TODO into Commander (here)
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

	c.submit_response(si)

	// TODO into Storage
	// session_save()

}
func (c *CommanderSingle) submit_response(s SessionInfo) {
	// 	TODO Port this function from ghes-mirva-server
}

func (c *CommanderSingle) collectRequestInfo(w http.ResponseWriter, r *http.Request, sessionId int) (string, []co.OwnerRepo, string, error) {
	slog.Debug("Collecting session info")

	if r.Body == nil {
		err := errors.New("Missing request body")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNoContent)
		return "", []co.OwnerRepo{}, "", err
	}
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		var w http.ResponseWriter
		slog.Error("Error reading MRVA submission body", "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []co.OwnerRepo{}, "", err
	}
	msg, err := TrySubmitMsg(buf)
	if err != nil {
		// Unknown message
		slog.Error("Unknown MRVA submission body format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []co.OwnerRepo{}, "", err
	}
	// Decompose the SubmitMsg and keep information

	// Save the query pack and keep the location
	if !isBase64Gzip([]byte(msg.QueryPack)) {
		slog.Error("MRVA submission body querypack has invalid format")
		err := errors.New("MRVA submission body querypack has invalid format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []co.OwnerRepo{}, "", err
	}
	session_tgz_ref, err := c.extract_tgz(msg.QueryPack, sessionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []co.OwnerRepo{}, "", err
	}

	// 2. Save the language
	session_language := msg.Language

	// 3. Save the repositories
	var session_repositories []co.OwnerRepo

	for _, v := range msg.Repositories {
		t := strings.Split(v, "/")
		if len(t) != 2 {
			slog.Error("Invalid owner / repository entry", "entry", t)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		session_repositories = append(session_repositories,
			co.OwnerRepo{Owner: t[0], Repo: t[1]})
	}
	return session_language, session_repositories, session_tgz_ref, nil
}

// Try to extract a SubmitMsg from a json-encoded buffer
func TrySubmitMsg(buf []byte) (SubmitMsg, error) {
	buf1 := make([]byte, len(buf))
	copy(buf1, buf)
	dec := json.NewDecoder(bytes.NewReader(buf1))
	dec.DisallowUnknownFields()
	var m SubmitMsg
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

func (c *CommanderSingle) extract_tgz(qp string, sessionID int) (string, error) {
	// These are decoded manually via
	//    base64 -d < foo1 | gunzip | tar t | head -20
	// base64 decode the body
	slog.Debug("Extracting query pack")

	tgz, err := base64.StdEncoding.DecodeString(qp)
	if err != nil {
		slog.Error("querypack body decoding error:", err)
		return "", err
	}

	session_query_pack_tgz_filepath, err := c.st.Storage.SaveQueryPack(tgz, sessionID)
	if err != nil {
		return "", err
	}

	return session_query_pack_tgz_filepath, err
}
