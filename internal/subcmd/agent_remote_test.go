package subcmd

import (
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestBuildSSHCommand(t *testing.T) {
	cfg := store.DefaultRigConfig("rig", "/tmp/repo")
	cfg.RemoteHost = "example.com"
	cfg.RemoteUser = "alice"
	cfg.RemotePort = 2222
	cmd, args := buildSSHCommand(cfg, []string{"tmux", "ls"}, true)
	if cmd != "ssh" {
		t.Fatalf("expected ssh")
	}
	if len(args) < 5 || args[0] != "-t" || args[1] != "-p" || args[2] != "2222" {
		t.Fatalf("unexpected args: %v", args)
	}
}
