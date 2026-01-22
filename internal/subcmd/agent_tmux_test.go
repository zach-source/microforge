//go:build tmux

package subcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func TestAgentSpawnWakeStopTmux(t *testing.T) {
	if os.Getenv("MF_ENABLE_TMUX_TESTS") != "1" {
		t.Skip("set MF_ENABLE_TMUX_TESTS=1 to run tmux integration tests")
	}
	if _, err := util.Run(nil, "tmux", "-V"); err != nil {
		t.Skip("tmux not available")
	}

	home := t.TempDir()
	rigName := "rig"
	rigDir := store.RigDir(home, rigName)
	if err := util.EnsureDir(rigDir); err != nil {
		t.Fatalf("ensure rig dir: %v", err)
	}
	cfg := store.RigConfig{
		Name:            rigName,
		RepoPath:        filepath.Join(home, "repo"),
		TmuxPrefix:      "mf",
		RuntimeProvider: "claude",
		RuntimeCmd:      "sleep",
		RuntimeArgs:     []string{"60"},
		RuntimeRoles:    map[string]store.RuntimeSpec{},
		CreatedAt:       store.MustNow(),
	}
	if err := store.SaveRigConfig(store.RigConfigPath(home, rigName), cfg); err != nil {
		t.Fatalf("save rig config: %v", err)
	}

	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	rigRow, err := store.EnsureRig(db, cfg)
	if err != nil {
		t.Fatalf("ensure rig: %v", err)
	}

	worktree := store.CellWorktreeDir(home, rigName, "payments")
	if err := util.EnsureDir(worktree); err != nil {
		t.Fatalf("ensure worktree: %v", err)
	}
	cell, err := store.CreateCell(db, rigRow.ID, "payments", "services/payments", worktree)
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	agent, err := store.CreateAgent(db, rigRow.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	_ = agent

	if err := Agent(home, []string{"spawn", rigName, cell.Name, "builder"}); err != nil {
		t.Fatalf("agent spawn: %v", err)
	}
	defer func() {
		_ = Agent(home, []string{"stop", rigName, cell.Name, "builder"})
	}()

	if err := Agent(home, []string{"wake", rigName, cell.Name, "builder"}); err != nil {
		t.Fatalf("agent wake: %v", err)
	}

	if err := Agent(home, []string{"stop", rigName, cell.Name, "builder"}); err != nil {
		t.Fatalf("agent stop: %v", err)
	}
}
