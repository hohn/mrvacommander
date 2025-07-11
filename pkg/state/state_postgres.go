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

	const createTable = `
	CREATE TABLE IF NOT EXISTS analyze_results (
		session_id INTEGER NOT NULL,
		owner TEXT NOT NULL,
		repo TEXT NOT NULL,
		result JSONB NOT NULL,
		PRIMARY KEY (session_id, owner, repo)
	);
	`

	_, err := pool.Exec(ctx, createTable)
	if err != nil {
		slog.Error("Failed to create analyze_results table", "error", err)
		os.Exit(1)
	}

	slog.Info("Schema initialized: analyze_results")
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
