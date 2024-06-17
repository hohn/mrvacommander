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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mrvacommander/pkg/common"
	"mrvacommander/pkg/storage"

	"github.com/gorilla/mux"
)

func setupEndpoints(c CommanderAPI) {
	r := mux.NewRouter()
	c.vis = st

	//
	// First are the API endpoints that mirror those used in the github API
	//
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses", c.MRVARequest)
	// 			  /repos/hohn   /mrva-controller/code-scanning/codeql/variant-analyses
	// Or via
	r.HandleFunc("/{repository_id}/code-scanning/codeql/variant-analyses", c.MRVARequestID)

	r.HandleFunc("/", c.RootHandler)

	// This is the standalone status request.
	// It's also the first request made when downloading; the difference is on the
	// client side's handling.
	r.HandleFunc("/repos/{owner}/{repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}", c.MRVAStatus)

	r.HandleFunc("/repos/{controller_owner}/{controller_repo}/code-scanning/codeql/variant-analyses/{codeql_variant_analysis_id}/repos/{repo_owner}/{repo_name}", c.MRVADownloadArtifact)

	// Not implemented:
	// r.HandleFunc("/codeql-query-console/codeql-variant-analysis-repo-tasks/{codeql_variant_analysis_id}/{repo_id}/{owner_id}/{controller_repo_id}", MRVADownLoad3)
	// r.HandleFunc("/github-codeql-query-console-prod/codeql-variant-analysis-repo-tasks/{codeql_variant_analysis_id}/{repo_id}", MRVADownLoad4)

	//
	// Now some support API endpoints
	//
	r.HandleFunc("/download-server/{local_path:.*}", c.MRVADownloadServe)

	//
	// Bind to a port and pass our router in
	//
	// TODO make this a configuration entry
	log.Fatal(http.ListenAndServe(":8080", r))
}

func (c *CommanderSingle) StatusResponse(w http.ResponseWriter, js common.JobSpec, ji common.JobInfo, vaid int) {
	slog.Debug("Submitting status response", "session", vaid)

	all_scanned := []common.ScannedRepo{}
	jobs := storage.GetJobList(js.JobID)
	for _, job := range jobs {
		astat := storage.GetStatus(js.JobID, job.NWO).ToExternalString()
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

	astat := storage.GetStatus(js.JobID, js.NameWithOwner).ToExternalString()

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
	spec := storage.GetJobList(id)
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

	ji := storage.GetJobInfo(js)

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

func (c *CommanderSingle) DownloadResponse(w http.ResponseWriter, js common.JobSpec, vaid int) {
	slog.Debug("Forming download response", "session", vaid, "job", js)

	astat := storage.GetStatus(vaid, js.NameWithOwner)

	var dlr common.DownloadResponse
	if astat == common.StatusSuccess {

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

func (c *CommanderSingle) MRVADownloadServe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("File download request", "local_path", vars["local_path"])

	FileDownload(w, vars["local_path"])
}

func FileDownload(w http.ResponseWriter, path string) {
	slog.Debug("Sending zip file with .sarif/.bqrs", "path", path)

	fpath, res, err := storage.ResultAsFile(path)
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
	slog.Info("New mrva using repository_id=%v\n", vars["repository_id"])
}

func (c *CommanderSingle) MRVARequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva run ", "owner", vars["owner"], "repo", vars["repo"])

	session_id := c.vis.ServerStore.NextID()
	session_owner := vars["owner"]
	session_controller_repo := vars["repo"]
	slog.Info("new run", "id: ", fmt.Sprint(session_id), session_owner, session_controller_repo)
	session_language, session_repositories, session_tgz_ref, err := c.collectRequestInfo(w, r, session_id)
	if err != nil {
		return
	}

	not_found_repos, analysisRepos := c.vis.ServerStore.FindAvailableDBs(session_repositories)

	c.vis.Queue.StartAnalyses(analysisRepos, session_id, session_language)

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

func nwoToNwoStringArray(nwo []common.NameWithOwner) ([]string, int) {
	repos := []string{}
	count := len(nwo)
	for _, repo := range nwo {
		repos = append(repos, fmt.Sprintf("%s/%s", repo.Owner, repo.Repo))
	}
	return repos, count
}

func submit_response(sn SessionInfo) ([]byte, error) {
	// Construct the response bottom-up
	var m_cr common.ControllerRepo
	var m_ac common.Actor

	repos, count := nwoToNwoStringArray(sn.NotFoundRepos)
	r_nfr := common.NotFoundRepos{RepositoryCount: count, RepositoryFullNames: repos}

	repos, count = nwoToNwoStringArray(sn.AccessMismatchRepos)
	r_amr := common.AccessMismatchRepos{RepositoryCount: count, Repositories: repos}

	repos, count = nwoToNwoStringArray(sn.NoCodeqlDBRepos)
	r_ncd := common.NoCodeqlDBRepos{RepositoryCount: count, Repositories: repos}

	// TODO fill these with real values?
	repos, count = nwoToNwoStringArray(sn.NoCodeqlDBRepos)
	r_olr := common.OverLimitRepos{RepositoryCount: count, Repositories: repos}

	m_skip := common.SkippedRepositories{
		AccessMismatchRepos: r_amr,
		NotFoundRepos:       r_nfr,
		NoCodeqlDBRepos:     r_ncd,
		OverLimitRepos:      r_olr}

	m_sr := common.SubmitResponse{
		Actor:               m_ac,
		ControllerRepo:      m_cr,
		ID:                  sn.ID,
		QueryLanguage:       sn.Language,
		QueryPackURL:        sn.QueryPack,
		CreatedAt:           time.Now().Format(time.RFC3339),
		UpdatedAt:           time.Now().Format(time.RFC3339),
		Status:              "in_progress",
		SkippedRepositories: m_skip,
	}

	// Store data needed later
	joblist := storage.GetJobList(sn.ID)

	for _, job := range joblist {
		storage.SetJobInfo(common.JobSpec{
			JobID:         sn.ID,
			NameWithOwner: job.NWO,
		}, common.JobInfo{
			QueryLanguage:       sn.Language,
			CreatedAt:           m_sr.CreatedAt,
			UpdatedAt:           m_sr.UpdatedAt,
			SkippedRepositories: m_skip,
		},
		)
	}

	// Encode the response as JSON
	submit_response, err := json.Marshal(m_sr)
	if err != nil {
		slog.Warn("Error encoding response as JSON:", err)
		return nil, err
	}
	return submit_response, nil

}

func (c *CommanderSingle) collectRequestInfo(w http.ResponseWriter, r *http.Request, sessionId int) (string, []common.NameWithOwner, string, error) {
	slog.Debug("Collecting session info")

	if r.Body == nil {
		err := errors.New("missing request body")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNoContent)
		return "", []common.NameWithOwner{}, "", err
	}
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		var w http.ResponseWriter
		slog.Error("Error reading MRVA submission body", "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, "", err
	}
	msg, err := TrySubmitMsg(buf)
	if err != nil {
		// Unknown message
		slog.Error("Unknown MRVA submission body format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, "", err
	}
	// Decompose the SubmitMsg and keep information

	// Save the query pack and keep the location
	if !isBase64Gzip([]byte(msg.QueryPack)) {
		slog.Error("MRVA submission body querypack has invalid format")
		err := errors.New("MRVA submission body querypack has invalid format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, "", err
	}
	session_tgz_ref, err := c.extract_tgz(msg.QueryPack, sessionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", []common.NameWithOwner{}, "", err
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

	session_query_pack_tgz_filepath, err := c.vis.ServerStore.SaveQueryPack(tgz, sessionID)
	if err != nil {
		return "", err
	}

	return session_query_pack_tgz_filepath, err
}
