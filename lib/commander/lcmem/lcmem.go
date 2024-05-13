// The in-memory implementation of the mrva commander library
package lcmem

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/advanced-security/mrvacommander/interfaces/mci"
	"github.com/gorilla/mux"
	"github.com/hohn/ghes-mirva-server/analyze"
	"github.com/hohn/ghes-mirva-server/api"
	co "github.com/hohn/ghes-mirva-server/common"
	"github.com/hohn/ghes-mirva-server/store"
)

type Commander struct {
}

func (c *Commander) Run(st mci.State) {
}

func (c *Commander) Setup(st mci.State) {
	r := mux.NewRouter()

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

func (c *Commander) StatusResponse(w http.ResponseWriter, js co.JobSpec, ji co.JobInfo, vaid int) {
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

func (c *Commander) RootHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Request on /")
}

func (c *Commander) MirvaStatus(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("No jobs found for given job id",
			"id", vars["codeql_variant_analysis_id"])
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	job := spec[0]

	js := co.JobSpec{
		ID:        job.QueryPackId,
		OwnerRepo: job.ORL,
	}

	ji := store.GetJobInfo(js)

	analyze.StatusResponse(w, js, ji, id)
	c.StatusResponse(w, js, ji, id)
}

// Download artifacts
func (c *Commander) MirvaDownloadArtifact(w http.ResponseWriter, r *http.Request) {
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
	analyze.DownloadResponse(w, js, vaid)

}

func (c *Commander) MirvaDownloadServe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("File download request", "local_path", vars["local_path"])

	analyze.FileDownload(w, vars["local_path"])
}

func (c *Commander) MirvaRequestID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva using repository_id=%v\n", vars["repository_id"])
}

func (c *Commander) MirvaRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slog.Info("New mrva run ", "owner", vars["owner"], "repo", vars["repo"])
	// TODO Change this to functional style?
	// session := new(MirvaSession)
	// session.id = next_id()
	// session.owner = vars["owner"]
	// session.controller_repo = vars["repo"]
	// session.collect_info(w, r)
	// session.find_available_DBs()
	// session.start_analyses()
	// session.submit_response(w)
	// session.save()
}
