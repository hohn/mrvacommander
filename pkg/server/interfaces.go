package server

import "net/http"

type Commander interface{}

type CommanderAPI interface {
	MirvaRequestID(w http.ResponseWriter, r *http.Request)
	MirvaRequest(w http.ResponseWriter, r *http.Request)
	RootHandler(w http.ResponseWriter, r *http.Request)
	MirvaStatus(w http.ResponseWriter, r *http.Request)
	MirvaDownloadArtifact(w http.ResponseWriter, r *http.Request)
	MirvaDownloadServe(w http.ResponseWriter, r *http.Request)
}
