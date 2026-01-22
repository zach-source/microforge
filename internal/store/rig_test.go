package store

import (
	"path/filepath"
	"testing"
)

func TestRigConfigRuntimeRoles(t *testing.T) {
	cfg := DefaultRigConfig("rig", "/tmp/repo")
	cfg.RuntimeRoles["builder"] = RuntimeSpec{Cmd: "custom", Args: []string{"--flag"}}
	path := filepath.Join(t.TempDir(), "rig.json")
	if err := SaveRigConfig(path, cfg); err != nil {
		t.Fatalf("save rig config: %v", err)
	}
	loaded, err := LoadRigConfig(path)
	if err != nil {
		t.Fatalf("load rig config: %v", err)
	}
	spec, ok := loaded.RuntimeRoles["builder"]
	if !ok || spec.Cmd != "custom" || len(spec.Args) != 1 {
		t.Fatalf("expected runtime role to load")
	}
}
