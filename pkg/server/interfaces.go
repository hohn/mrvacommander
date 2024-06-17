package server

import "net/http"

type Commander interface{}

type CommanderAPI interface {
	MRVARequestID(w http.ResponseWriter, r *http.Request)
	MRVARequest(w http.ResponseWriter, r *http.Request)
	RootHandler(w http.ResponseWriter, r *http.Request)
	MRVAStatus(w http.ResponseWriter, r *http.Request)
	MRVADownloadArtifact(w http.ResponseWriter, r *http.Request)
	MRVADownloadServe(w http.ResponseWriter, r *http.Request)
}
