package subcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetActiveAgent(t *testing.T) {
	worktree := t.TempDir()
	mfDir := filepath.Join(worktree, ".mf")
	if err := os.MkdirAll(mfDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	builder := []byte(`{"role":"builder"}`)
	reviewer := []byte(`{"role":"reviewer"}`)
	if err := os.WriteFile(filepath.Join(mfDir, "active-agent-builder.json"), builder, 0o644); err != nil {
		t.Fatalf("write builder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mfDir, "active-agent-reviewer.json"), reviewer, 0o644); err != nil {
		t.Fatalf("write reviewer: %v", err)
	}

	if err := setActiveAgent(worktree, "reviewer"); err != nil {
		t.Fatalf("set active: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(mfDir, "active-agent.json"))
	if err != nil {
		t.Fatalf("read active: %v", err)
	}
	if string(b) != string(reviewer) {
		t.Fatalf("expected reviewer active agent")
	}
}
