package subcmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/microforge/internal/library"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func TestLibraryQueryCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(library.QueryResponse{Results: []library.Result{{Service: "svc", Path: "p", Snippet: "s"}}, Source: "local"})
	}))
	defer server.Close()

	home := t.TempDir()
	rigDir := rig.RigDir(home, "rig")
	if err := util.EnsureDir(rigDir); err != nil {
		t.Fatalf("ensure rig dir: %v", err)
	}

	cfg := rig.DefaultRigConfig("rig", "/tmp/repo")
	cfg.LibraryAddr = server.Listener.Addr().String()
	if err := rig.SaveRigConfig(rig.RigConfigPath(home, "rig"), cfg); err != nil {
		t.Fatalf("save rig config: %v", err)
	}

	if err := Library(home, []string{"query", "rig", "--q", "hello"}); err != nil {
		t.Fatalf("library query: %v", err)
	}
}
