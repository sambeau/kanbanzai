// Package coordination provides the coordination server components.
package coordination

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DDL statements for the coordination server schema.
const (
	ddlCounters = `
CREATE TABLE IF NOT EXISTS counters (
    project_id  TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    next_value  INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (project_id, entity_type)
);`

	ddlBatchFeatureSeqs = `
CREATE TABLE IF NOT EXISTS batch_feature_seqs (
    project_id TEXT NOT NULL,
    batch_id   TEXT NOT NULL,
    next_seq   INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (project_id, batch_id)
);`

	ddlAllocations = `
CREATE TABLE IF NOT EXISTS allocations (
    project_id   TEXT NOT NULL,
    entity_type  TEXT NOT NULL,
    slug         TEXT NOT NULL,
    allocated_id TEXT NOT NULL,
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, entity_type, slug)
);`

	ddlAllocateID = `
CREATE OR REPLACE FUNCTION allocate_id(
    p_project_id  TEXT,
    p_entity_type TEXT,
    p_prefix      TEXT,
    p_slug        TEXT
) RETURNS TEXT AS $$
DECLARE
    existing TEXT;
    next_val INTEGER;
    result_id TEXT;
BEGIN
    SELECT allocated_id INTO existing
    FROM allocations
    WHERE project_id = p_project_id
      AND entity_type = p_entity_type
      AND slug = p_slug;
    IF existing IS NOT NULL THEN
        RETURN existing;
    END IF;

    INSERT INTO counters (project_id, entity_type, next_value)
    VALUES (p_project_id, p_entity_type, 2)
    ON CONFLICT (project_id, entity_type)
    DO UPDATE SET next_value = counters.next_value + 1
    RETURNING next_value - 1 INTO next_val;

    result_id := p_prefix || next_val || '-' || p_slug;
    INSERT INTO allocations (project_id, entity_type, slug, allocated_id)
    VALUES (p_project_id, p_entity_type, p_slug, result_id);

    RETURN result_id;
END;
$$ LANGUAGE plpgsql;`

	allocateIDQuery       = `SELECT allocate_id($1, $2, $3, $4)`
	allocateFeatureSeqSQL = `
INSERT INTO batch_feature_seqs (project_id, batch_id, next_seq)
VALUES ($1, $2, 2)
ON CONFLICT (project_id, batch_id)
DO UPDATE SET next_seq = batch_feature_seqs.next_seq + 1
RETURNING next_seq - 1`
)

// DB wraps a pgxpool.Pool for the coordination server's Postgres connection.
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new DB by connecting to the given database URL. On failure
// (unreachable host, invalid credentials, etc.) the error is returned — the
// caller decides fallback behaviour. TLS is enabled by default (pgx
// behaviour).
func New(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("coordination: creating pool: %w", err)
	}
	// Verify connectivity before returning so callers get immediate
	// feedback on misconfiguration.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("coordination: connecting to database: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Close closes the connection pool. Callers should ensure no queries are in
// flight before calling Close.
func (db *DB) Close() {
	db.pool.Close()
}

// Ping verifies that the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("coordination: ping: %w", err)
	}
	return nil
}

// Migrate runs the coordination server DDL, ensuring all tables and the
// allocate_id function exist. It is idempotent — safe to run multiple times.
func (db *DB) Migrate(ctx context.Context) error {
	for _, ddl := range []string{ddlCounters, ddlBatchFeatureSeqs, ddlAllocations, ddlAllocateID} {
		if _, err := db.pool.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("coordination: migrate: %w", err)
		}
	}
	return nil
}

// AllocateID atomically allocates a unique ID for the given project, entity
// type, and slug. The returned ID has the format {prefix}{n}-{slug} where n
// is a monotonically increasing counter per (project_id, entity_type).
// Repeated calls with the same arguments return the same ID.
func (db *DB) AllocateID(ctx context.Context, projectID, entityType, prefix, slug string) (string, error) {
	var id string
	if err := db.pool.QueryRow(ctx, allocateIDQuery,
		projectID, entityType, prefix, slug,
	).Scan(&id); err != nil {
		return "", fmt.Errorf("coordination: allocate id: %w", err)
	}
	return id, nil
}

// AllocateFeatureSeq atomically increments and returns the next sequence
// number for features within a batch. The first call for a given
// (project_id, batch_id) returns 1.
func (db *DB) AllocateFeatureSeq(ctx context.Context, projectID, batchID string) (int, error) {
	var seq int
	if err := db.pool.QueryRow(ctx, allocateFeatureSeqSQL,
		projectID, batchID,
	).Scan(&seq); err != nil {
		return 0, fmt.Errorf("coordination: allocate feature seq: %w", err)
	}
	return seq, nil
}
