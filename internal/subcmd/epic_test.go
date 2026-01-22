package subcmd

import (
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestEpicCreateAddStatus(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	task, err := store.CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := Epic(home, []string{"create", "rig", "--title", "Hardening"}); err != nil {
		t.Fatalf("epic create: %v", err)
	}
	row := db.QueryRow("SELECT id FROM epics LIMIT 1")
	var epicID string
	if err := row.Scan(&epicID); err != nil {
		t.Fatalf("scan epic: %v", err)
	}

	if err := Epic(home, []string{"add-task", "rig", "--epic", epicID, "--task", task.ID}); err != nil {
		t.Fatalf("epic add-task: %v", err)
	}

	if err := Epic(home, []string{"status", "rig", "--epic", epicID}); err != nil {
		t.Fatalf("epic status: %v", err)
	}
}

func TestTaskSplitCreatesLinks(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	_, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	_, err = store.CreateCell(db, rig.ID, "billing", "services/billing", filepath.Join(home, "worktree2"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	parent, err := store.CreateTask(db, rig.ID, "improve", "Parent", "Body", "services/payments")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := Task(home, []string{"split", "rig", "--task", parent.ID, "--cells", "payments,billing"}); err != nil {
		t.Fatalf("task split: %v", err)
	}

	row := db.QueryRow("SELECT COUNT(1) FROM task_links WHERE parent_task_id = ?", parent.ID)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan task links: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 child links, got %d", count)
	}
}

func TestEpicAssignAndClose(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	_, err = store.CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := store.CreateTask(db, rig.ID, "review", "Review task", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	epic, err := store.CreateEpic(db, rig.ID, "Epic", "Body")
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	if err := store.AddTaskToEpic(db, epic.ID, task.ID); err != nil {
		t.Fatalf("add task: %v", err)
	}

	if err := Epic(home, []string{"assign", "rig", "--epic", epic.ID}); err != nil {
		t.Fatalf("epic assign: %v", err)
	}
	if err := store.MarkTaskDone(db, task.ID); err != nil {
		t.Fatalf("mark task done: %v", err)
	}
	if err := Epic(home, []string{"close", "rig", "--epic", epic.ID}); err != nil {
		t.Fatalf("epic close: %v", err)
	}
	got, err := store.GetEpic(db, epic.ID)
	if err != nil {
		t.Fatalf("get epic: %v", err)
	}
	if got.Status != "closed" {
		t.Fatalf("expected closed epic, got %s", got.Status)
	}
}

func TestEpicConflictCreatesRequest(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	_, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	epic, err := store.CreateEpic(db, rig.ID, "Epic", "Body")
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	if err := Epic(home, []string{"conflict", "rig", "--epic", epic.ID, "--cell", "payments", "--details", "conflict"}); err != nil {
		t.Fatalf("epic conflict: %v", err)
	}
	row := db.QueryRow("SELECT COUNT(1) FROM requests")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan requests: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 request, got %d", count)
	}
}
