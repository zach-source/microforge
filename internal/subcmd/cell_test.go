package subcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func setupRigForCell(t *testing.T) (home string) {
	t.Helper()
	home = t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := util.EnsureDir(store.RigDir(home, "rig")); err != nil {
		t.Fatalf("ensure rig dir: %v", err)
	}
	cfg := store.DefaultRigConfig("rig", repo)
	if err := store.SaveRigConfig(store.RigConfigPath(home, "rig"), cfg); err != nil {
		t.Fatalf("save rig config: %v", err)
	}
	db, err := store.OpenDB(store.DBPath(home, "rig"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := store.EnsureRig(db, cfg); err != nil {
		t.Fatalf("ensure rig: %v", err)
	}
	return home
}

func TestCellAddAndBootstrap(t *testing.T) {
	home := setupRigForCell(t)

	if err := Cell(home, []string{"add", "rig", "payments", "--scope", "services/payments"}); err != nil {
		t.Fatalf("cell add: %v", err)
	}
	if err := Cell(home, []string{"bootstrap", "rig", "payments"}); err != nil {
		t.Fatalf("cell bootstrap: %v", err)
	}

	wt := store.CellWorktreeDir(home, "rig", "payments")
	if _, err := os.Stat(filepath.Join(wt, ".mf", "active-agent.json")); err != nil {
		t.Fatalf("missing active-agent.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wt, ".claude", "settings.json")); err != nil {
		t.Fatalf("missing settings.json: %v", err)
	}

	settings, err := os.ReadFile(filepath.Join(wt, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if !strings.Contains(string(settings), "mforge hook stop --role builder") {
		t.Fatalf("expected builder stop hook")
	}
}

func TestCellBootstrapWithArchitect(t *testing.T) {
	home := setupRigForCell(t)

	if err := Cell(home, []string{"add", "rig", "payments", "--scope", "services/payments"}); err != nil {
		t.Fatalf("cell add: %v", err)
	}
	if err := Cell(home, []string{"bootstrap", "rig", "payments", "--architect"}); err != nil {
		t.Fatalf("cell bootstrap: %v", err)
	}

	wt := store.CellWorktreeDir(home, "rig", "payments")
	settings, err := os.ReadFile(filepath.Join(wt, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if !strings.Contains(string(settings), "mforge hook stop --role architect") {
		t.Fatalf("expected architect stop hook")
	}
}

func TestCellAddMissingScope(t *testing.T) {
	home := setupRigForCell(t)
	if err := Cell(home, []string{"add", "rig", "payments"}); err == nil {
		t.Fatalf("expected error for missing --scope")
	}
}
