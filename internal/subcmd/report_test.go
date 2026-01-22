package subcmd

import (
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestReportCommand(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	_, err = store.CreateRequest(db, rig.ID, cell.ID, "monitor", "high", "p1", cell.ScopePrefix, "{}")
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	_, err = store.CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := Report(home, []string{"rig"}); err != nil {
		t.Fatalf("report: %v", err)
	}
	if err := Report(home, []string{"rig", "--cell", "payments"}); err != nil {
		t.Fatalf("report cell: %v", err)
	}
}
