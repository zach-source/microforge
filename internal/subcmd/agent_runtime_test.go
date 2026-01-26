package subcmd

import (
	"testing"

	"github.com/example/microforge/internal/rig"
)

func TestRuntimeForRole(t *testing.T) {
	cfg := rig.DefaultRigConfig("rig", "/tmp/repo")
	cfg.RuntimeRoles["builder"] = rig.RuntimeSpec{Cmd: "custom", Args: []string{"--x"}}
	cmd, args := runtimeForRole(cfg, "builder")
	if cmd != "custom" || len(args) < 1 {
		t.Fatalf("expected role override")
	}
	if !containsArg(args, "--dangerously-skip-permissions") {
		t.Fatalf("expected dangerous skip permissions flag")
	}
	cmd, args = runtimeForRole(cfg, "reviewer")
	if cmd != cfg.RuntimeCmd || len(args) != len(cfg.RuntimeArgs) {
		t.Fatalf("expected default runtime")
	}
	if !containsArg(args, "--dangerously-skip-permissions") {
		t.Fatalf("expected dangerous skip permissions flag")
	}
}

func containsArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}
