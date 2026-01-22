package subcmd

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func setupRigForSubcmd(t *testing.T) (home string, rig store.RigRow, db *sql.DB) {
	t.Helper()
	home = t.TempDir()
	rigName := "rig"
	if err := util.EnsureDir(filepath.Dir(store.DBPath(home, rigName))); err != nil {
		t.Fatalf("ensure rig dir: %v", err)
	}
	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	cfg := store.RigConfig{
		Name:            rigName,
		RepoPath:        filepath.Join(home, "repo"),
		TmuxPrefix:      "mf",
		RuntimeProvider: "claude",
		RuntimeCmd:      "claude",
		RuntimeArgs:     []string{"--resume"},
		RuntimeRoles:    map[string]store.RuntimeSpec{},
		CreatedAt:       store.MustNow(),
	}
	rig, err = store.EnsureRig(db, cfg)
	if err != nil {
		t.Fatalf("ensure rig: %v", err)
	}
	return home, rig, db
}

func TestTaskCreateAndList(t *testing.T) {
	home, _, db := setupRigForSubcmd(t)
	defer db.Close()

	args := []string{"create", "rig", "--title", "Add /healthz", "--body", "Body", "--scope", "services/payments", "--kind", "fix"}
	if err := Task(home, args); err != nil {
		t.Fatalf("task create: %v", err)
	}

	rows, err := db.Query("SELECT id, kind, title, scope_prefix, status FROM tasks")
	if err != nil {
		t.Fatalf("query tasks: %v", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, kind, title, status string
		var scope sql.NullString
		if err := rows.Scan(&id, &kind, &title, &scope, &status); err != nil {
			t.Fatalf("scan task: %v", err)
		}
		if kind != "fix" || title == "" || status != "created" {
			t.Fatalf("unexpected task fields: kind=%s title=%s status=%s", kind, title, status)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 task, got %d", count)
	}

	if err := Task(home, []string{"list", "rig"}); err != nil {
		t.Fatalf("task list: %v", err)
	}
}

func TestTaskCreateMissingTitle(t *testing.T) {
	home, _, db := setupRigForSubcmd(t)
	defer db.Close()

	args := []string{"create", "rig"}
	if err := Task(home, args); err == nil {
		t.Fatalf("expected error for missing title")
	}
}
