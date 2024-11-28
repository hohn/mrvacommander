/*
   dependencies
   go get -u golang.org/x/exp/slog

   on-the-fly
   go run bin/hepc-serve.go --codeql-db-dir  db-collection-py-1

   compiled
   cd ~/work-gh/mrva/mrvacommander/client/qldbtools/
   go build -o ./bin/hepc-serve.bin ./bin/hepc-serve.go

   test
   curl http://127.0.0.1:8080/api/v1/latest_results/codeql-all -o foo
   curl $(head -1 foo | jq  -r ".result_url" |sed 's|hepc|127.0.0.1:8080/db|g;') -o foo.zip

*/
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/exp/slog"
)

var dbDir string

func serveFile(w http.ResponseWriter, r *http.Request) {
	fullPath := r.URL.Path[len("/db/"):]

	resolvedPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		slog.Warn("failed to resolve symlink", slog.String("fullPath", fullPath),
			slog.String("error", err.Error()))
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if fileInfo, err := os.Stat(resolvedPath); err != nil || fileInfo.IsDir() {
		slog.Warn("file not found or is a directory", slog.String("resolvedPath", resolvedPath))
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	slog.Info("serving file", slog.String("resolvedPath", resolvedPath))
	http.ServeFile(w, r, resolvedPath)
}

func serveMetadata(w http.ResponseWriter, r *http.Request) {
	metadataPath := filepath.Join(dbDir, "metadata.json")
	if fileInfo, err := os.Stat(metadataPath); err != nil || fileInfo.IsDir() {
		slog.Warn("metadata.json not found", slog.String("metadataPath", metadataPath))
		http.Error(w, "metadata.json not found", http.StatusNotFound)
		return
	}

	slog.Info("serving metadata.json", slog.String("metadataPath", metadataPath))
	http.ServeFile(w, r, metadataPath)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("incoming request", slog.String("method", r.Method), slog.String("url", r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

func main() {
	var host string
	var port int

	flag.StringVar(&dbDir, "codeql-db-dir", "", "Directory containing CodeQL database files (required)")
	flag.StringVar(&host, "host", "127.0.0.1", "Host address for the HTTP server")
	flag.IntVar(&port, "port", 8080, "Port for the HTTP server")
	flag.Parse()

	if dbDir == "" {
		slog.Error("missing required flag", slog.String("flag", "--codeql-db-dir"))
		os.Exit(1)
	}

	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		slog.Error("invalid directory", slog.String("dbDir", dbDir))
		os.Exit(1)
	}

	slog.Info("starting server", slog.String("host", host), slog.Int("port", port), slog.String("dbDir", dbDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/db/", serveFile)
	mux.HandleFunc("/index", serveMetadata)
	mux.HandleFunc("/api/v1/latest_results/codeql-all", serveMetadata)

	loggedHandler := logMiddleware(mux)

	addr := fmt.Sprintf("%s:%d", host, port)
	slog.Info("server listening", slog.String("address", addr))
	if err := http.ListenAndServe(addr, loggedHandler); err != nil {
		slog.Error("server error", slog.String("error", err.Error()))
	}
}
