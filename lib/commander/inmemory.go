package analyze

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/hohn/ghes-mirva-server/api"
	co "github.com/hohn/ghes-mirva-server/common"
	"github.com/hohn/ghes-mirva-server/store"
)

func StatusResponse(w http.ResponseWriter, js co.JobSpec, ji co.JobInfo, vaid int) {
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
