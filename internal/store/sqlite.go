package store

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil { return nil, err }
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil { db.Close(); return nil, err }
	return db, nil
}

func migrate(db *sql.DB) error {
	for _, s := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	} {
		if _, err := db.Exec(s); err != nil { return err }
	}

	schema := `
CREATE TABLE IF NOT EXISTS rigs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  repo_path TEXT NOT NULL,
  tmux_prefix TEXT NOT NULL,
  runtime_cmd TEXT NOT NULL,
  runtime_args_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cells (
  id TEXT PRIMARY KEY,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  scope_prefix TEXT NOT NULL,
  worktree_path TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(rig_id, name)
);

CREATE TABLE IF NOT EXISTS agents (
  id TEXT PRIMARY KEY,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  cell_id TEXT NOT NULL REFERENCES cells(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  name TEXT NOT NULL,
  tmux_session TEXT NOT NULL,
  status TEXT NOT NULL,
  last_seen_at TEXT,
  created_at TEXT NOT NULL,
  UNIQUE(cell_id, role)
);

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  scope_prefix TEXT,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assignments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  inbox_relpath TEXT NOT NULL,
  outbox_relpath TEXT NOT NULL,
  completion_promise TEXT NOT NULL,
  requested_by_agent_id TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assignments_agent_status ON assignments(agent_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_rig_status ON tasks(rig_id, status);
`
	_, err := db.Exec(schema)
	return err
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	if ctx == nil { ctx = context.Background() }
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil { return err }
	if err := fn(tx); err != nil { tx.Rollback(); return err }
	return tx.Commit()
}

func MustNow() string { return time.Now().UTC().Format(time.RFC3339) }
