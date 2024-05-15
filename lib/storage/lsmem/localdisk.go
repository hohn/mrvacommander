package lsmem

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/advanced-security/mrvacommander/types/tsto"
	co "github.com/hohn/ghes-mirva-server/common"
)

type Storage struct {
	CurrentID int
}

func (s *Storage) NextID() int {
	s.CurrentID += 1
	return s.CurrentID
}

func (s *Storage) SaveQueryPack(tgz []byte, sessionId int) (string, error) {
	// Save the tar.gz body
	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("No working directory")
		panic(err)
	}

	dirpath := path.Join(cwd, "var", "codeql", "querypacks")
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		slog.Error("Unable to create query pack output directory",
			"dir", dirpath)
		return "", err
	}

	fpath := path.Join(dirpath, fmt.Sprintf("qp-%d.tgz", sessionId))
	err = os.WriteFile(fpath, tgz, 0644)
	if err != nil {
		slog.Error("unable to save querypack body decoding error", "path", fpath)
		return "", err
	} else {
		slog.Info("Query pack saved to ", "path", fpath)
	}

	return fpath, nil
}

//		Determine for which repositories codeql databases are available.
//
//	 Those will be the analysis_repos.  The rest will be skipped.
func (s *Storage) FindAvailableDBs(analysisReposRequested []co.OwnerRepo) (not_found_repos []co.OwnerRepo,
	analysisRepos *map[co.OwnerRepo]tsto.DBLocation) {
	slog.Debug("Looking for available CodeQL databases")

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("No working directory")
		return
	}

	analysisRepos = &map[co.OwnerRepo]tsto.DBLocation{}

	not_found_repos = []co.OwnerRepo{}

	for _, rep := range analysisReposRequested {
		dbPrefix := filepath.Join(cwd, "codeql", "dbs", rep.Owner, rep.Repo)
		dbName := fmt.Sprintf("%s_%s_db.zip", rep.Owner, rep.Repo)
		dbPath := filepath.Join(dbPrefix, dbName)

		if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
			slog.Info("Database does not exist for repository ", "owner/repo", rep,
				"path", dbPath)
			not_found_repos = append(not_found_repos, rep)
		} else {
			slog.Info("Found database for ", "owner/repo", rep, "path", dbPath)
			(*analysisRepos)[rep] = tsto.DBLocation{Prefix: dbPrefix, File: dbName}
		}
	}
	return not_found_repos, analysisRepos
}
