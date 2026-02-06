// Package state provides agent state persistence via SQLite.
//
// This is the single-source agent state database at ~/.orch/state.db.
// It replaces the distributed JOIN across 6 systems (OpenCode, beads, tmux,
// registry, workspace, Anthropic API) with fast local reads (~1ms).
//
// Uses modernc.org/sqlite (pure Go, no CGO) with WAL mode for concurrent reads.
//
// See: .kb/investigations/2026-02-06-design-single-source-agent-state.md
package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// DefaultDBPath returns the default path for the state database.
// Location: ~/.orch/state.db
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".orch", "state.db")
}

// DB wraps a SQLite database connection for agent state.
type DB struct {
	db   *sql.DB
	path string
	mu   sync.Mutex // protects writes (reads are concurrent via WAL)
}

// Open opens (or creates) the state database at the given path.
// Enables WAL mode for concurrent reads and creates the schema if needed.
func Open(path string) (*DB, error) {
	if path == "" {
		path = DefaultDBPath()
	}
	if path == "" {
		return nil, fmt.Errorf("could not determine state database path")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory %s: %w", dir, err)
	}

	// Open with modernc.org/sqlite driver
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open state database: %w", err)
	}

	// Enable WAL mode for concurrent reads (daemon + serve + status)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set busy timeout to handle concurrent access gracefully (5 seconds)
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	sdb := &DB{db: db, path: path}

	// Create schema if needed
	if err := sdb.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return sdb, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.db == nil {
		return nil
	}
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

// createSchema creates the agents table and indexes if they don't exist.
func (d *DB) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS agents (
		-- Core identity (set at spawn, immutable)
		workspace_name  TEXT PRIMARY KEY,
		beads_id        TEXT UNIQUE,
		session_id      TEXT,
		tmux_window     TEXT,
		mode            TEXT NOT NULL DEFAULT 'opencode',
		skill           TEXT,
		model           TEXT,
		tier            TEXT,
		project_dir     TEXT NOT NULL,
		project_name    TEXT,
		spawn_time      INTEGER NOT NULL,
		git_baseline    TEXT,
		issue_title     TEXT,
		issue_type      TEXT,
		issue_priority  INTEGER,

		-- Mutable lifecycle state (event-driven writes)
		phase           TEXT,
		phase_summary   TEXT,
		phase_reported_at INTEGER,
		is_processing   INTEGER DEFAULT 0,
		session_updated_at INTEGER,
		is_completed    INTEGER DEFAULT 0,
		is_abandoned    INTEGER DEFAULT 0,
		completed_at    INTEGER,
		abandoned_at    INTEGER,

		-- Token aggregates (updated by periodic poll during processing)
		tokens_input    INTEGER DEFAULT 0,
		tokens_output   INTEGER DEFAULT 0,
		tokens_reasoning INTEGER DEFAULT 0,
		tokens_cache_read INTEGER DEFAULT 0,
		tokens_total    INTEGER DEFAULT 0,

		-- Timestamps
		created_at      INTEGER NOT NULL,
		updated_at      INTEGER NOT NULL
	);

	-- Indexes for common query patterns
	CREATE INDEX IF NOT EXISTS idx_agents_beads_id ON agents(beads_id);
	CREATE INDEX IF NOT EXISTS idx_agents_session_id ON agents(session_id);
	CREATE INDEX IF NOT EXISTS idx_agents_project ON agents(project_name);
	CREATE INDEX IF NOT EXISTS idx_agents_active ON agents(is_completed, is_abandoned);
	CREATE INDEX IF NOT EXISTS idx_agents_phase ON agents(phase);
	`

	_, err := d.db.Exec(schema)
	return err
}
