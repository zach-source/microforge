package subcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestMonitorRunTestsCreatesRequestOnFailure(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	worktree := filepath.Join(home, "worktree")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	_, err := store.CreateCell(db, rig.ID, "payments", "services/payments", worktree)
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}

	args := []string{"run-tests", "rig", "payments", "--cmd", "go", "list", "./does/not/exist"}
	if err := Monitor(home, args); err != nil {
		t.Fatalf("monitor run-tests: %v", err)
	}

	var count int
	row := db.QueryRow("SELECT COUNT(1) FROM requests")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan requests: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 request, got %d", count)
	}
}

func TestMonitorRunTestsNoRequestOnSuccess(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	worktree := filepath.Join(home, "worktree")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	_, err := store.CreateCell(db, rig.ID, "payments", "services/payments", worktree)
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}

	args := []string{"run-tests", "rig", "payments", "--cmd", "go", "env", "GOROOT"}
	if err := Monitor(home, args); err != nil {
		t.Fatalf("monitor run-tests: %v", err)
	}

	var count int
	row := db.QueryRow("SELECT COUNT(1) FROM requests")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan requests: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 requests, got %d", count)
	}
}
