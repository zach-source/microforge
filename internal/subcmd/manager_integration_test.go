package subcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestManagerReconcileIntegration(t *testing.T) {
	home, rig, db, worktree := setupRigDB(t)
	defer db.Close()

	if err := os.MkdirAll(filepath.Join(worktree, "mail", "inbox"), 0o755); err != nil {
		t.Fatalf("mkdir inbox: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(worktree, "mail", "outbox"), 0o755); err != nil {
		t.Fatalf("mkdir outbox: %v", err)
	}

	task, err := store.CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", "services/payments")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	cell, err := store.GetCell(db, rig.ID, "payments")
	if err != nil {
		t.Fatalf("get cell: %v", err)
	}
	agent, err := store.GetAgentByCellRole(db, cell.ID, "builder")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}

	promise := "PROMISE"
	inboxRel := filepath.Join("mail", "inbox", task.ID+".md")
	outboxRel := filepath.Join("mail", "outbox", task.ID+".md")
	_, err = store.CreateAssignment(db, rig.ID, task.ID, agent.ID, inboxRel, outboxRel, promise, nil)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, inboxRel), []byte("inbox"), 0o644); err != nil {
		t.Fatalf("write inbox: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, outboxRel), []byte("done "+promise), 0o644); err != nil {
		t.Fatalf("write outbox: %v", err)
	}

	updated, err := reconcile(home, rig.Name)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 update, got %d", updated)
	}

	archiveOutbox := filepath.Join(worktree, "mail", "archive", filepath.Base(outboxRel))
	if _, err := os.Stat(archiveOutbox); err != nil {
		t.Fatalf("expected archived outbox: %v", err)
	}
}
