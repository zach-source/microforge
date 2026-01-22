package subcmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func TestRequestCreateListTriage(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	_, err = store.CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	payload, _ := json.Marshal(map[string]string{"title": "Fix failing tests", "body": "Details"})
	args := []string{"create", "rig", "--cell", "payments", "--role", "monitor", "--severity", "high", "--priority", "p1", "--scope", "services/payments", "--payload", string(payload)}
	if err := Request(home, args); err != nil {
		t.Fatalf("request create: %v", err)
	}

	if err := Request(home, []string{"list", "rig"}); err != nil {
		t.Fatalf("request list: %v", err)
	}

	var reqID string
	row := db.QueryRow("SELECT id FROM requests LIMIT 1")
	if err := row.Scan(&reqID); err != nil {
		t.Fatalf("scan request: %v", err)
	}
	if reqID == "" {
		t.Fatalf("expected request id")
	}

	triageArgs := []string{"triage", "rig", "--request", reqID, "--action", "create-task"}
	if err := Request(home, triageArgs); err != nil {
		t.Fatalf("request triage: %v", err)
	}

	var count int
	row = db.QueryRow("SELECT COUNT(1) FROM tasks")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 task, got %d", count)
	}
}

func TestRequestCreateArchitect(t *testing.T) {
	home, rig, db := setupRigForSubcmd(t)
	defer db.Close()

	_, err := store.CreateCell(db, rig.ID, "payments", "services/payments", filepath.Join(home, "worktree"))
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}

	args := []string{"create", "rig", "--cell", "payments", "--role", "architect", "--severity", "med", "--priority", "p2", "--payload", "{}"}
	if err := Request(home, args); err != nil {
		t.Fatalf("request create: %v", err)
	}
}
