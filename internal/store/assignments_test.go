package store

import (
	"path/filepath"
	"testing"
)

func TestClaimNextAssignmentForAgent(t *testing.T) {
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
	agent, err := CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	_, err = CreateAssignment(db, rig.ID, task.ID, agent.ID, "mail/inbox/a.md", "mail/outbox/a.md", "DONE", nil)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	assn, ok, err := ClaimNextAssignmentForAgent(db, agent.ID)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if !ok || assn.Status != "running" {
		t.Fatalf("expected running assignment")
	}

	_, ok, err = ClaimNextAssignmentForAgent(db, agent.ID)
	if err != nil {
		t.Fatalf("claim again: %v", err)
	}
	if ok {
		t.Fatalf("expected no queued assignments")
	}
}
