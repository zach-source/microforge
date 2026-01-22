package subcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestInitCreatesRigConfigAndDB(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	if err := Init(home, []string{"rig", "--repo", repo}); err != nil {
		t.Fatalf("init: %v", err)
	}

	cfgPath := store.RigConfigPath(home, "rig")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected rig config: %v", err)
	}
	if _, err := os.Stat(store.DBPath(home, "rig")); err != nil {
		t.Fatalf("expected db: %v", err)
	}
}

func TestInitMissingRepo(t *testing.T) {
	home := t.TempDir()
	if err := Init(home, []string{"rig"}); err == nil {
		t.Fatalf("expected error for missing --repo")
	}
}
