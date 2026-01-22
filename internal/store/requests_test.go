package store

import (
	"path/filepath"
	"testing"
)

func TestRequestCRUD(t *testing.T) {
	tmp := t.TempDir()
	db, err := OpenDB(filepath.Join(tmp, "mf.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	cfg := DefaultRigConfig("rig", filepath.Join(tmp, "repo"))
	rig, err := EnsureRig(db, cfg)
	if err != nil {
		t.Fatalf("ensure rig: %v", err)
	}
	cell, err := CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(tmp, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}

	r, err := CreateRequest(db, rig.ID, cell.ID, "monitor", "high", "p1", cell.ScopePrefix, `{"title":"Failing tests"}`)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if r.Status != "new" {
		t.Fatalf("expected status new, got %s", r.Status)
	}

	list, err := ListRequests(db, rig.ID, &cell.ID, nil, nil)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 request, got %d", len(list))
	}

	if err := UpdateRequestStatus(db, r.ID, "triaged"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, err := GetRequest(db, r.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status != "triaged" {
		t.Fatalf("expected status triaged, got %s", got.Status)
	}
}
