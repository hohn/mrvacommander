package server

import "net/http"

type CommanderAPI interface {
	MRVARequestID(w http.ResponseWriter, r *http.Request)
	MRVARequest(w http.ResponseWriter, r *http.Request)
	RootHandler(w http.ResponseWriter, r *http.Request)
	MRVAStatusID(w http.ResponseWriter, r *http.Request)
	MRVAStatus(w http.ResponseWriter, r *http.Request)
	MRVADownloadArtifactID(w http.ResponseWriter, r *http.Request)
	MRVADownloadArtifact(w http.ResponseWriter, r *http.Request)
	MRVADownloadServe(w http.ResponseWriter, r *http.Request)
}
