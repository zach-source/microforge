package hooks

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func setupHookDB(t *testing.T) (*sql.DB, store.RigRow, store.CellRow, store.AgentRow, string) {
	t.Helper()
	tmp := t.TempDir()
	db, err := store.OpenDB(filepath.Join(tmp, "mf.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	cfg := store.RigConfig{
		Name:            "rig",
		RepoPath:        filepath.Join(tmp, "repo"),
		TmuxPrefix:      "mf",
		RuntimeProvider: "claude",
		RuntimeCmd:      "claude",
		RuntimeArgs:     []string{"--resume"},
		RuntimeRoles:    map[string]store.RuntimeSpec{},
		CreatedAt:       store.MustNow(),
	}
	rig, err := store.EnsureRig(db, cfg)
	if err != nil {
		t.Fatalf("ensure rig: %v", err)
	}
	worktree := filepath.Join(tmp, "worktree")
	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", worktree)
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	agent, err := store.CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return db, rig, cell, agent, worktree
}

func TestStopHookNoAssignments(t *testing.T) {
	db, rig, cell, agent, worktree := setupHookDB(t)
	defer db.Close()

	identity := AgentIdentity{
		RigName:  rig.Name,
		DBPath:   filepath.Join(worktree, "..", "mf.db"),
		CellName: cell.Name,
		Role:     "builder",
		AgentID:  agent.ID,
		Scope:    cell.ScopePrefix,
		Worktree: worktree,
	}

	resp, err := StopHook(context.Background(), db, identity)
	if err != nil {
		t.Fatalf("stop hook: %v", err)
	}
	if resp.Continue {
		t.Fatalf("expected Continue=false")
	}
}
