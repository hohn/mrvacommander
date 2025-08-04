package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/hohn/mrvacommander/pkg/common"
	"github.com/hohn/mrvacommander/pkg/queue"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ----- PGState holds the shared connection pool
type PGState struct {
	pool *pgxpool.Pool
}

func validateEnvVars(requiredEnvVars []string) {
	missing := false

	for _, envVar := range requiredEnvVars {
		if _, ok := os.LookupEnv(envVar); !ok {
			slog.Error("Missing required environment variable", "key", envVar)
			missing = true
		}
	}

	if missing {
		os.Exit(1)
	}
}

func NewPGState() *PGState {
	ctx := context.Background()

	required := []string{
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		// Host & port may be omitted if you rely on Docker DNS, but list
		// them here to make the requirement explicit:
		"POSTGRES_HOST",
		"POSTGRES_PORT",
	}

	validateEnvVars(required)

	// Assemble from vars
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, db)
	slog.Info("Assembled Postgres connection URL from POSTGRES_* variables", "url", dbURL)

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("Failed to parse connection URL", "url", dbURL, "error", err)
		os.Exit(1)
	}

	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("Failed to create pgx pool", "error", err)
		os.Exit(1)
	}

	slog.Info("Connected to Postgres", "max_conns", config.MaxConns)

	SetupSchemas(pool)

	return &PGState{pool: pool}
}

func SetupSchemas(pool *pgxpool.Pool) {
	ctx := context.Background()

	schemas := []struct {
		name string
		sql  string
	}{
		{
			name: "job_repo_map",
			sql: `
				CREATE TABLE IF NOT EXISTS job_repo_map (
					job_repo_id SERIAL PRIMARY KEY,
					owner       TEXT NOT NULL,
					repo        TEXT NOT NULL,
					UNIQUE(owner, repo)
				);
			  `,
		},
		{
			name: "session_id_seq",
			sql: `
        		CREATE SEQUENCE IF NOT EXISTS session_id_seq;
	        `,
		},
		{
			name: "analyze_results",
			sql: `
				CREATE TABLE IF NOT EXISTS analyze_results (
					session_id INTEGER NOT NULL,
					owner TEXT NOT NULL,
					repo TEXT NOT NULL,
					result JSONB NOT NULL,
					PRIMARY KEY (session_id, owner, repo)
				);
			`,
		},
		{
			name: "analyze_jobs",
			sql: `
				CREATE TABLE IF NOT EXISTS analyze_jobs (
					session_id INTEGER NOT NULL,
					owner TEXT NOT NULL,
					repo TEXT NOT NULL,
					payload JSONB NOT NULL,
					PRIMARY KEY (session_id, owner, repo)
				);
			`,
		},
		{
			name: "job_info",
			sql: `
				CREATE TABLE IF NOT EXISTS job_info (
					session_id INTEGER NOT NULL,
					owner TEXT NOT NULL,
					repo TEXT NOT NULL,
					payload JSONB NOT NULL,
					PRIMARY KEY (session_id, owner, repo)
				);
			`,
		},
		{
			name: "job_status",
			sql: `
				CREATE TABLE IF NOT EXISTS job_status (
					session_id INTEGER NOT NULL,
					owner TEXT NOT NULL,
					repo TEXT NOT NULL,
					status INTEGER NOT NULL,
					PRIMARY KEY (session_id, owner, repo)
				);
			`,
		},
	}

	for _, schema := range schemas {
		_, err := pool.Exec(ctx, schema.sql)
		if err != nil {
			slog.Error("Failed to create table", "table", schema.name, "error", err)
			os.Exit(1)
		}
		slog.Info("Schema initialized", "table", schema.name)
	}
}

// ----- Sequence-based NextID (implements ServerState)
func (s *PGState) NextID() int {
	ctx := context.Background()

	var id int
	err := s.pool.QueryRow(ctx, `SELECT nextval('session_id_seq')`).Scan(&id)
	if err != nil {
		slog.Error("NextID query failed", "error", err)
		panic("NextID(): " + err.Error()) // interface doesn't allow returning error
	}

	slog.Debug("NextID generated", "id", id)
	return id
}

func (s *PGState) SetResult(js common.JobSpec, ar queue.AnalyzeResult) {
	ctx := context.Background()

	ar.Spec = js // ensure internal consistency

	jsonBytes, err := json.Marshal(ar)
	if err != nil {
		slog.Error("SetResult: JSON marshal failed", "job", js, "error", err)
		panic("SetResult(): " + err.Error())
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO analyze_results (session_id, owner, repo, result)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id, owner, repo)
		DO UPDATE SET result = EXCLUDED.result
	`, js.SessionID, js.Owner, js.Repo, jsonBytes)

	if err != nil {
		slog.Error("SetResult: insert/update failed", "job", js, "error", err)
		panic("SetResult(): " + err.Error())
	}
}

func (s *PGState) GetResult(js common.JobSpec) (queue.AnalyzeResult, error) {
	ctx := context.Background()

	var jsonBytes []byte
	err := s.pool.QueryRow(ctx, `
        SELECT result FROM analyze_results
        WHERE session_id = $1 AND owner = $2 AND repo = $3
    `, js.SessionID, js.Owner, js.Repo).Scan(&jsonBytes)
	if err != nil {
		return queue.AnalyzeResult{}, err
	}

	var ar queue.AnalyzeResult
	if err := json.Unmarshal(jsonBytes, &ar); err != nil {
		return queue.AnalyzeResult{}, fmt.Errorf("unmarshal AnalyzeResult: %w", err)
	}

	return ar, nil
}

func (s *PGState) SetJobInfo(js common.JobSpec, ji common.JobInfo) {
	ctx := context.Background()

	jiJSON, err := json.Marshal(ji)
	if err != nil {
		slog.Error("SetJobInfo: marshal failed", "job", js, "error", err)
		panic("SetJobInfo(): " + err.Error())
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO job_info (session_id, owner, repo, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id, owner, repo)
		DO UPDATE SET payload = EXCLUDED.payload
	`, js.SessionID, js.Owner, js.Repo, jiJSON)

	if err != nil {
		slog.Error("SetJobInfo: insert/update failed", "job", js, "error", err)
		panic("SetJobInfo(): " + err.Error())
	}
}

func (s *PGState) GetJobInfo(js common.JobSpec) (common.JobInfo, error) {
	ctx := context.Background()

	var jsonBytes []byte
	err := s.pool.QueryRow(ctx, `
		SELECT payload FROM job_info
		WHERE session_id = $1 AND owner = $2 AND repo = $3
	`, js.SessionID, js.Owner, js.Repo).Scan(&jsonBytes)
	if err != nil {
		return common.JobInfo{}, err
	}

	var ji common.JobInfo
	if err := json.Unmarshal(jsonBytes, &ji); err != nil {
		return common.JobInfo{}, fmt.Errorf("unmarshal JobInfo: %w", err)
	}

	return ji, nil
}

func (s *PGState) SetStatus(js common.JobSpec, status common.Status) {
	ctx := context.Background()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_status (session_id, owner, repo, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id, owner, repo)
		DO UPDATE SET status = EXCLUDED.status
	`, js.SessionID, js.Owner, js.Repo, status)

	if err != nil {
		slog.Error("SetStatus failed", "job", js, "status", status, "error", err)
		panic("SetStatus(): " + err.Error())
	}
}

func (s *PGState) GetStatus(js common.JobSpec) (common.Status, error) {
	ctx := context.Background()

	var status int
	err := s.pool.QueryRow(ctx, `
		SELECT status
		FROM job_status
		WHERE session_id = $1 AND owner = $2 AND repo = $3
	`, js.SessionID, js.Owner, js.Repo).Scan(&status)
	if err != nil {
		return 0, err // caller must interpret not-found vs. real error
	}

	return common.Status(status), nil
}

// GetRepoId returns a stable unique ID for a given (owner, repo).
// If the pair doesn't exist, it is inserted atomically.
func (s *PGState) GetRepoId(cno common.NameWithOwner) int {
	ctx := context.Background()

	var jobRepoID int

	err := s.pool.QueryRow(ctx, `
			INSERT INTO job_repo_map (owner, repo)
			VALUES ($1, $2)
			ON CONFLICT (owner, repo) DO UPDATE SET owner = EXCLUDED.owner
			RETURNING job_repo_id
		`, cno.Owner, cno.Repo).Scan(&jobRepoID)
	if err != nil {
		slog.Error("GetRepoId failed", "NameWithOwner", cno, "error", err)
		panic("GetRepoId: " + err.Error())
	}

	return jobRepoID

}

func (s *PGState) AddJob(job queue.AnalyzeJob) {
	ctx := context.Background()
	js := job.Spec

	// Begin transaction for atomic operation
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("AddJob: failed to begin transaction", "job", js, "error", err)
		panic("AddJob(): " + err.Error())
	}
	defer tx.Rollback(ctx) // Will be ignored if tx.Commit() succeeds

	// 1. Store AnalyzeJob payload -------------------------------
	jb, err := json.Marshal(job)
	if err != nil {
		slog.Error("AddJob: marshal failed", "job", js, "error", err)
		panic("AddJob(): " + err.Error())
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO analyze_jobs (session_id, owner, repo, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, js.SessionID, js.Owner, js.Repo, jb)
	if err != nil {
		slog.Error("AddJob: insert analyze_jobs failed", "job", js, "error", err)
		panic("AddJob(): " + err.Error())
	}

	// 2. Get job_repo_id
	jobRepoID := s.GetRepoId(job.Spec.NameWithOwner)

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		slog.Error("AddJob: failed to commit transaction", "job", js, "error", err)
		panic("AddJob(): " + err.Error())
	}

	slog.Debug("AddJob stored", "session", js.SessionID, "jobRepoId", jobRepoID, "owner", js.Owner, "repo", js.Repo)
}

func (s *PGState) GetJobList(sessionId int) ([]queue.AnalyzeJob, error) {
	ctx := context.Background()

	rows, err := s.pool.Query(ctx, `
		SELECT payload FROM analyze_jobs
		WHERE session_id = $1
		ORDER BY owner, repo
	`, sessionId)
	if err != nil {
		slog.Error("GetJobList: query failed", "session_id", sessionId, "error", err)
		return nil, err
	}
	defer rows.Close()

	var jobs []queue.AnalyzeJob
	for rows.Next() {
		var jsonBytes []byte
		if err := rows.Scan(&jsonBytes); err != nil {
			slog.Error("GetJobList: scan failed", "error", err)
			return nil, err
		}
		var job queue.AnalyzeJob
		if err := json.Unmarshal(jsonBytes, &job); err != nil {
			slog.Error("GetJobList: unmarshal failed", "error", err)
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		slog.Error("GetJobList: rows iteration failed", "error", err)
		return nil, err
	}

	return jobs, nil
}

func (s *PGState) GetJobSpecByRepoId(sessionId, jobRepoId int) (common.JobSpec, error) {
	ctx := context.Background()

	var owner, repo string
	err := s.pool.QueryRow(ctx, `
		SELECT owner, repo
		FROM job_repo_map
		WHERE job_repo_id = $1
	`, jobRepoId).Scan(&owner, &repo)
	if err != nil {
		return common.JobSpec{}, err
	}
	return common.JobSpec{
		SessionID: sessionId,
		NameWithOwner: common.NameWithOwner{
			Owner: owner,
			Repo:  repo,
		},
	}, nil
}
