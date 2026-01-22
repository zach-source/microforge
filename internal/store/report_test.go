package store

import (
	"path/filepath"
	"testing"
)

func TestCountsAndOldest(t *testing.T) {
	tmp := t.TempDir()
	db, err := OpenDB(filepath.Join(tmp, "mf.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	cfg := DefaultRigConfig("rig", filepath.Join(tmp, "repo"))
	rig, err := EnsureRig(db, cfg)
	if err != nil {
		t.Fatalf("ensure rig: %v", err)
	}
	cell, err := CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(tmp, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	_, err = CreateRequest(db, rig.ID, cell.ID, "monitor", "high", "p1", cell.ScopePrefix, "{}")
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	_, err = CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	reqCounts, err := CountRequestsByStatus(db, rig.ID, &cell.ID)
	if err != nil {
		t.Fatalf("count requests: %v", err)
	}
	if reqCounts["new"] != 1 {
		t.Fatalf("expected 1 request")
	}
	taskCounts, err := CountTasksByStatus(db, rig.ID, cell.ScopePrefix)
	if err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if taskCounts["created"] != 1 {
		t.Fatalf("expected 1 task")
	}
	if _, err := OldestRequestCreatedAt(db, rig.ID, &cell.ID); err != nil {
		t.Fatalf("oldest request: %v", err)
	}
	if _, err := OldestTaskCreatedAt(db, rig.ID, cell.ScopePrefix); err != nil {
		t.Fatalf("oldest task: %v", err)
	}
}
