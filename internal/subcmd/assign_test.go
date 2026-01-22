package subcmd

import (
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestAssignCreatesAssignment(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	agent, err := store.CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := store.CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	args := []string{"rig", "--task", task.ID, "--cell", "payments", "--role", "builder", "--promise", "PROMISE"}
	if err := Assign(home, args); err != nil {
		t.Fatalf("assign: %v", err)
	}

	var count int
	row := db.QueryRow("SELECT COUNT(1) FROM assignments WHERE task_id = ? AND agent_id = ?", task.ID, agent.ID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan assignments: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 assignment, got %d", count)
	}

	var status string
	row = db.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID)
	if err := row.Scan(&status); err != nil {
		t.Fatalf("scan task: %v", err)
	}
	if status != "assigned" {
		t.Fatalf("expected task assigned, got %s", status)
	}
}

func TestAssignMissingArgs(t *testing.T) {
	home, _, db := setupRigForSubcmd(t)
	defer db.Close()

	if err := Assign(home, []string{"rig"}); err == nil {
		t.Fatalf("expected error for missing args")
	}

	if err := Assign(home, []string{"rig", "--task", "x"}); err == nil {
		t.Fatalf("expected error for missing args")
	}
}

func TestAssignDefaultPromise(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	agent, err := store.CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := store.CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	args := []string{"rig", "--task", task.ID, "--cell", "payments", "--role", "builder"}
	if err := Assign(home, args); err != nil {
		t.Fatalf("assign: %v", err)
	}

	var promise string
	row := db.QueryRow("SELECT completion_promise FROM assignments WHERE task_id = ? AND agent_id = ?", task.ID, agent.ID)
	if err := row.Scan(&promise); err != nil {
		t.Fatalf("scan assignment: %v", err)
	}
	if promise != "DONE" {
		t.Fatalf("expected default promise DONE, got %s", promise)
	}
}
