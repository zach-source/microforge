package store

import (
	"path/filepath"
	"testing"
)

func TestRigConfigRemoteFields(t *testing.T) {
	cfg := DefaultRigConfig("rig", "/tmp/repo")
	cfg.RemoteHost = "example.com"
	cfg.RemoteUser = "alice"
	cfg.RemotePort = 2222
	cfg.RemoteWorkdir = "/work/{cell}"
	cfg.LibraryDocs = []string{"docs"}
	path := filepath.Join(t.TempDir(), "rig.json")
	if err := SaveRigConfig(path, cfg); err != nil {
		t.Fatalf("save rig config: %v", err)
	}
	loaded, err := LoadRigConfig(path)
	if err != nil {
		t.Fatalf("load rig config: %v", err)
	}
	if loaded.RemoteHost != "example.com" || loaded.RemotePort != 2222 {
		t.Fatalf("remote fields not loaded")
	}
	if len(loaded.LibraryDocs) != 1 {
		t.Fatalf("library docs not loaded")
	}
}
