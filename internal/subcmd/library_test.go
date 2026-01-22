package subcmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/microforge/internal/library"
	"github.com/example/microforge/internal/store"
)

func TestLibraryQueryCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(library.QueryResponse{Results: []library.Result{{Service: "svc", Path: "p", Snippet: "s"}}, Source: "local"})
	}))
	defer server.Close()

	home, _, db := setupRigForSubcmd(t)
	defer db.Close()

	cfg := store.DefaultRigConfig("rig", "/tmp/repo")
	cfg.LibraryAddr = server.Listener.Addr().String()
	if err := store.SaveRigConfig(store.RigConfigPath(home, "rig"), cfg); err != nil {
		t.Fatalf("save rig config: %v", err)
	}

	if err := Library(home, []string{"query", "rig", "--q", "hello"}); err != nil {
		t.Fatalf("library query: %v", err)
	}
}
