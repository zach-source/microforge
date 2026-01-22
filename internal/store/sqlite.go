package store

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	for _, s := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	} {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TEXT NOT NULL
);`); err != nil {
		return err
	}
	var current int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&current); err != nil {
		return err
	}
	type migration struct {
		id   int
		name string
		up   []string
	}
	migrations := []migration{
		{
			id:   1,
			name: "base_schema",
			up: []string{
				`CREATE TABLE IF NOT EXISTS rigs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  repo_path TEXT NOT NULL,
  tmux_prefix TEXT NOT NULL,
  runtime_cmd TEXT NOT NULL,
  runtime_args_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);`,
				`CREATE TABLE IF NOT EXISTS cells (
  id TEXT PRIMARY KEY,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  scope_prefix TEXT NOT NULL,
  worktree_path TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(rig_id, name)
);`,
				`CREATE TABLE IF NOT EXISTS agents (
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
);`,
				`CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  scope_prefix TEXT,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);`,
				`CREATE TABLE IF NOT EXISTS assignments (
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
);`,
				`CREATE INDEX IF NOT EXISTS idx_assignments_agent_status ON assignments(agent_id, status);`,
				`CREATE INDEX IF NOT EXISTS idx_tasks_rig_status ON tasks(rig_id, status);`,
			},
		},
		{
			id:   2,
			name: "requests",
			up: []string{
				`CREATE TABLE IF NOT EXISTS requests (
  id TEXT PRIMARY KEY,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  cell_id TEXT NOT NULL REFERENCES cells(id) ON DELETE CASCADE,
  source_role TEXT NOT NULL,
  severity TEXT NOT NULL,
  priority TEXT NOT NULL,
  scope_prefix TEXT,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);`,
				`CREATE INDEX IF NOT EXISTS idx_requests_rig_status ON requests(rig_id, status);`,
				`CREATE INDEX IF NOT EXISTS idx_requests_cell_status ON requests(cell_id, status);`,
				`CREATE INDEX IF NOT EXISTS idx_requests_priority_status ON requests(priority, status);`,
			},
		},
		{
			id:   3,
			name: "epics",
			up: []string{
				`CREATE TABLE IF NOT EXISTS epics (
  id TEXT PRIMARY KEY,
  rig_id TEXT NOT NULL REFERENCES rigs(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);`,
				`CREATE TABLE IF NOT EXISTS epic_tasks (
  epic_id TEXT NOT NULL REFERENCES epics(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY(epic_id, task_id)
);`,
				`CREATE TABLE IF NOT EXISTS task_links (
  parent_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  child_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY(parent_task_id, child_task_id)
);`,
			},
		},
		{
			id:   4,
			name: "runtime_provider",
			up: []string{
				`ALTER TABLE rigs ADD COLUMN runtime_provider TEXT NOT NULL DEFAULT 'claude';`,
				`ALTER TABLE rigs ADD COLUMN runtime_roles_json TEXT NOT NULL DEFAULT '{}';`,
			},
		},
		{
			id:   5,
			name: "remote_library",
			up: []string{
				`ALTER TABLE rigs ADD COLUMN remote_host TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE rigs ADD COLUMN remote_user TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE rigs ADD COLUMN remote_port INTEGER NOT NULL DEFAULT 22;`,
				`ALTER TABLE rigs ADD COLUMN remote_workdir TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE rigs ADD COLUMN remote_tmux_prefix TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE rigs ADD COLUMN library_addr TEXT NOT NULL DEFAULT '127.0.0.1:7331';`,
				`ALTER TABLE rigs ADD COLUMN library_docs_json TEXT NOT NULL DEFAULT '[]';`,
				`ALTER TABLE rigs ADD COLUMN library_context7_url TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE rigs ADD COLUMN library_context7_token TEXT NOT NULL DEFAULT '';`,
			},
		},
	}
	for _, m := range migrations {
		if m.id <= current {
			continue
		}
		if err := WithTx(context.Background(), db, func(tx *sql.Tx) error {
			for _, stmt := range m.up {
				if _, err := tx.Exec(stmt); err != nil {
					return err
				}
			}
			_, err := tx.Exec("INSERT INTO schema_migrations(version, name, applied_at) VALUES(?,?,?)", m.id, m.name, MustNow())
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func MustNow() string { return time.Now().UTC().Format(time.RFC3339) }
