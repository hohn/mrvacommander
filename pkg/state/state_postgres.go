package state

import (
	"context"
	"log/slog"
	"os"

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
	return &PGState{pool: pool}
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
