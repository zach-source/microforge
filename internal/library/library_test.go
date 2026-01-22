package library

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLibraryQueryLocal(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "docs", "serviceA")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p, "readme.md"), []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	lib, err := New(Config{Docs: []string{filepath.Join(tmp, "docs")}})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	resp, err := lib.Query("serviceA", "hello")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result")
	}
}

func TestLibraryContext7Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(QueryResponse{Results: []Result{{Service: "svc", Path: "remote", Snippet: "ok"}}, Source: "context7"})
	}))
	defer server.Close()

	lib, err := New(Config{Docs: []string{}, Context7URL: server.URL})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	resp, err := lib.Query("svc", "missing")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if resp.Source != "context7" {
		t.Fatalf("expected context7 source")
	}
}
