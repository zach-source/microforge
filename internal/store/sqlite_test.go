package store

import (
	"path/filepath"
	"testing"
)

func TestOpenDBCreatesSchema(t *testing.T) {
	tmp := t.TempDir()
	db, err := OpenDB(filepath.Join(tmp, "mf.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	want := map[string]bool{
		"rigs":              false,
		"cells":             false,
		"agents":            false,
		"tasks":             false,
		"assignments":       false,
		"requests":          false,
		"epics":             false,
		"epic_tasks":        false,
		"task_links":        false,
		"schema_migrations": false,
	}
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("query schema: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for table, found := range want {
		if !found {
			t.Fatalf("missing table: %s", table)
		}
	}

	row := db.QueryRow("SELECT MAX(version) FROM schema_migrations")
	var version int
	if err := row.Scan(&version); err != nil {
		t.Fatalf("scan migrations: %v", err)
	}
	if version < 5 {
		t.Fatalf("expected migrations applied, got version %d", version)
	}
}
