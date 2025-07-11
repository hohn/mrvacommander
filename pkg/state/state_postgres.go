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

// ----- Create a new PGState, reading connection info from env vars
func NewPGState() *PGState {
	ctx := context.Background()

	// Get connection URL from environment (standard: PGHOST, PGPORT, etc.)
	dbURL := os.Getenv("DATABASE_URL") // canonical way: use DATABASE_URL
	if dbURL == "" {
		slog.Error("DATABASE_URL environment variable must be set")
		os.Exit(1)
	}

	// Create a pool config
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("Failed to parse DATABASE_URL", "error", err)
		os.Exit(1)
	}

	// Optionally set pool tuning here
	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("Failed to create pgx pool", "error", err)
		os.Exit(1)
	}

	slog.Info("Connected to Postgres", "max_conns", config.MaxConns)

	// schema initialization
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

func (s *PGState) AddJob(job queue.AnalyzeJob) {
	ctx := context.Background()
	js := job.Spec

	jobJSON, err := json.Marshal(job)
	if err != nil {
		slog.Error("AddJob: marshal failed", "job", js, "error", err)
		panic("AddJob(): " + err.Error())
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO analyze_jobs (session_id, owner, repo, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, js.SessionID, js.Owner, js.Repo, jobJSON)

	if err != nil {
		slog.Error("AddJob: insert failed", "job", js, "error", err)
		panic("AddJob(): " + err.Error())
	}
}

func (s *PGState) GetJobList(sessionId int) ([]queue.AnalyzeJob, error) {
	ctx := context.Background()

	rows, err := s.pool.Query(ctx, `
		SELECT payload FROM analyze_jobs
		WHERE session_id = $1
	`, sessionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []queue.AnalyzeJob
	for rows.Next() {
		var jsonBytes []byte
		if err := rows.Scan(&jsonBytes); err != nil {
			return nil, err
		}
		var job queue.AnalyzeJob
		if err := json.Unmarshal(jsonBytes, &job); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}
