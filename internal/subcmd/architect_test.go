package subcmd

import (
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestArchitectCreatesRequest(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	_, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}

	args := []string{"docs", "rig", "--cell", "payments", "--details", "Update API docs"}
	if err := Architect(home, args); err != nil {
		t.Fatalf("architect docs: %v", err)
	}

	var count int
	row := db.QueryRow("SELECT COUNT(1) FROM requests WHERE source_role = 'architect'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan requests: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 architect request, got %d", count)
	}
}
