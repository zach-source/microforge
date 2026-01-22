package subcmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func setupRigDB(t *testing.T) (home string, rig store.RigRow, db *sql.DB, worktree string) {
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
	worktree = filepath.Join(home, "worktree")
	if err := os.MkdirAll(filepath.Join(worktree, "mail", "outbox"), 0o755); err != nil {
		t.Fatalf("mkdir outbox: %v", err)
	}
	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", worktree)
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	_, err = store.CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return home, rig, db, worktree
}

func TestManagerReconcileMarksDone(t *testing.T) {
	home, rig, db, worktree := setupRigDB(t)
	defer db.Close()

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
	outboxRel := filepath.Join("mail", "outbox", task.ID+".md")
	inboxRel := filepath.Join("mail", "inbox", task.ID+".md")
	_, err = store.CreateAssignment(db, rig.ID, task.ID, agent.ID, inboxRel, outboxRel, promise, nil)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(worktree, "mail", "inbox"), 0o755); err != nil {
		t.Fatalf("mkdir inbox: %v", err)
	}
	inboxAbs := filepath.Join(worktree, inboxRel)
	if err := os.WriteFile(inboxAbs, []byte("inbox"), 0o644); err != nil {
		t.Fatalf("write inbox: %v", err)
	}
	outAbs := filepath.Join(worktree, outboxRel)
	if err := os.WriteFile(outAbs, []byte("done: "+promise), 0o644); err != nil {
		t.Fatalf("write outbox: %v", err)
	}

	updated, err := reconcile(home, rig.Name)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 update, got %d", updated)
	}

	var status string
	row := db.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID)
	if err := row.Scan(&status); err != nil {
		t.Fatalf("scan task: %v", err)
	}
	if status != "done" {
		t.Fatalf("expected task done, got %s", status)
	}

	archiveInbox := filepath.Join(worktree, "mail", "archive", filepath.Base(inboxRel))
	if _, err := os.Stat(archiveInbox); err != nil {
		t.Fatalf("expected archived inbox: %v", err)
	}
	archiveOutbox := filepath.Join(worktree, "mail", "archive", filepath.Base(outboxRel))
	if _, err := os.Stat(archiveOutbox); err != nil {
		t.Fatalf("expected archived outbox: %v", err)
	}
}

func TestManagerReconcileSkipsWithoutPromise(t *testing.T) {
	home, rig, db, worktree := setupRigDB(t)
	defer db.Close()

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
	outboxRel := filepath.Join("mail", "outbox", task.ID+".md")
	_, err = store.CreateAssignment(db, rig.ID, task.ID, agent.ID, filepath.Join("mail", "inbox", task.ID+".md"), outboxRel, promise, nil)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	outAbs := filepath.Join(worktree, outboxRel)
	if err := os.WriteFile(outAbs, []byte("missing promise"), 0o644); err != nil {
		t.Fatalf("write outbox: %v", err)
	}

	updated, err := reconcile(home, rig.Name)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if updated != 0 {
		t.Fatalf("expected 0 updates, got %d", updated)
	}

	var status string
	row := db.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID)
	if err := row.Scan(&status); err != nil {
		t.Fatalf("scan task: %v", err)
	}
	if status == "done" {
		t.Fatalf("expected task not done")
	}
}

func TestManagerReconcileSkipsMissingOutbox(t *testing.T) {
	home, rig, db, _ := setupRigDB(t)
	defer db.Close()

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

	outboxRel := filepath.Join("mail", "outbox", task.ID+".md")
	_, err = store.CreateAssignment(db, rig.ID, task.ID, agent.ID, filepath.Join("mail", "inbox", task.ID+".md"), outboxRel, "PROMISE", nil)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	updated, err := reconcile(home, rig.Name)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if updated != 0 {
		t.Fatalf("expected 0 updates, got %d", updated)
	}
}
