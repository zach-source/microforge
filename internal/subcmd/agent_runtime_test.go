package subcmd

import (
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestRuntimeForRole(t *testing.T) {
	cfg := store.DefaultRigConfig("rig", "/tmp/repo")
	cfg.RuntimeRoles["builder"] = store.RuntimeSpec{Cmd: "custom", Args: []string{"--x"}}
	cmd, args := runtimeForRole(cfg, "builder")
	if cmd != "custom" || len(args) != 1 {
		t.Fatalf("expected role override")
	}
	cmd, args = runtimeForRole(cfg, "reviewer")
	if cmd != cfg.RuntimeCmd || len(args) != len(cfg.RuntimeArgs) {
		t.Fatalf("expected default runtime")
	}
}
