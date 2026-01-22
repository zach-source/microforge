package hooks

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/microforge/internal/store"
)

func setupRig(t *testing.T) (*sql.DB, store.RigRow, string) {
	t.Helper()
	tmp := t.TempDir()
	db, err := store.OpenDB(filepath.Join(tmp, "mf.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	cfg := store.RigConfig{
		Name:            "rig",
		RepoPath:        filepath.Join(tmp, "repo"),
		TmuxPrefix:      "mf",
		RuntimeProvider: "claude",
		RuntimeCmd:      "claude",
		RuntimeArgs:     []string{"--resume"},
		RuntimeRoles:    map[string]store.RuntimeSpec{},
		CreatedAt:       store.MustNow(),
	}
	rig, err := store.EnsureRig(db, cfg)
	if err != nil {
		t.Fatalf("ensure rig: %v", err)
	}
	return db, rig, tmp
}

func TestStopHookClaimsAssignmentAndWritesInbox(t *testing.T) {
	db, rig, tmp := setupRig(t)
	defer db.Close()

	worktree := filepath.Join(tmp, "worktree")
	if err := os.MkdirAll(filepath.Join(worktree, "mail", "inbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(worktree, "mail", "outbox"), 0o755); err != nil {
		t.Fatal(err)
	}

	cell, err := store.CreateCell(db, rig.ID, "payments", "services/payments", worktree)
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	agent, err := store.CreateAgent(db, rig.ID, cell.ID, "builder", "payments-builder", "mf-rig-payments-builder")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := store.CreateTask(db, rig.ID, "improve", "Add /healthz", "Body", cell.ScopePrefix)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	inboxRel := filepath.Join("mail", "inbox", task.ID+".md")
	outboxRel := filepath.Join("mail", "outbox", task.ID+".md")
	assn, err := store.CreateAssignment(db, rig.ID, task.ID, agent.ID, inboxRel, outboxRel, "PROMISE", nil)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	identity := AgentIdentity{
		RigName:  rig.Name,
		DBPath:   filepath.Join(tmp, "mf.db"),
		CellName: cell.Name,
		Role:     "builder",
		AgentID:  agent.ID,
		Scope:    cell.ScopePrefix,
		Worktree: worktree,
		Inbox:    "mail/inbox",
		Outbox:   "mail/outbox",
		Archive:  "mail/archive",
	}

	resp, err := StopHook(context.Background(), db, identity)
	if err != nil {
		t.Fatalf("stop hook: %v", err)
	}
	if !resp.Continue {
		t.Fatalf("expected Continue=true")
	}

	inboxAbs := filepath.Join(worktree, inboxRel)
	b, err := os.ReadFile(inboxAbs)
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}
	if string(b) == "" {
		t.Fatalf("expected inbox content")
	}

	var status string
	row := db.QueryRow("SELECT status FROM assignments WHERE id = ?", assn.ID)
	if err := row.Scan(&status); err != nil {
		t.Fatalf("scan assignment: %v", err)
	}
	if status != "running" {
		t.Fatalf("expected assignment running, got %s", status)
	}

	row = db.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID)
	if err := row.Scan(&status); err != nil {
		t.Fatalf("scan task: %v", err)
	}
	if status != "assigned" {
		t.Fatalf("expected task assigned, got %s", status)
	}
}

func TestGuardrailsHook(t *testing.T) {
	worktree := t.TempDir()
	identity := AgentIdentity{
		Role:     "builder",
		Scope:    "services/payments",
		Worktree: worktree,
	}
	in := ClaudeHookInput{ToolName: "Write", ToolInput: map[string]any{"path": "services/payments/main.go"}}

	t.Setenv("CLAUDE_TOOL_INPUT_FILE_PATH", "")

	resp, err := GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "allow" {
		t.Fatalf("expected allow, got %s", resp.Decision)
	}

	in.ToolInput = map[string]any{"path": "services/other/main.go"}
	resp, err = GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "deny" {
		t.Fatalf("expected deny, got %s", resp.Decision)
	}

	identity.Role = "monitor"
	resp, err = GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "deny" {
		t.Fatalf("expected deny for monitor, got %s", resp.Decision)
	}

	identity.Role = "architect"
	resp, err = GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "deny" {
		t.Fatalf("expected deny for architect, got %s", resp.Decision)
	}
}

func TestGuardrailsHookPathTraversalBlocked(t *testing.T) {
	worktree := t.TempDir()
	identity := AgentIdentity{
		Role:     "builder",
		Scope:    "services/payments",
		Worktree: worktree,
	}
	in := ClaudeHookInput{ToolName: "Write", ToolInput: map[string]any{"path": "services/payments/../../secrets.txt"}}

	t.Setenv("CLAUDE_TOOL_INPUT_FILE_PATH", "")

	resp, err := GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "deny" {
		t.Fatalf("expected deny for traversal, got %s", resp.Decision)
	}
}

func TestGuardrailsHookAbsolutePathAllowed(t *testing.T) {
	worktree := t.TempDir()
	identity := AgentIdentity{
		Role:     "builder",
		Scope:    "services/payments",
		Worktree: worktree,
	}
	inScope := filepath.Join(worktree, "services", "payments", "main.go")
	in := ClaudeHookInput{ToolName: "Write", ToolInput: map[string]any{"path": inScope}}

	t.Setenv("CLAUDE_TOOL_INPUT_FILE_PATH", "")

	resp, err := GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "allow" {
		t.Fatalf("expected allow for absolute path, got %s", resp.Decision)
	}
}

func TestGuardrailsHookEnvPathOverrides(t *testing.T) {
	worktree := t.TempDir()
	identity := AgentIdentity{
		Role:     "builder",
		Scope:    "services/payments",
		Worktree: worktree,
	}
	in := ClaudeHookInput{ToolName: "Write", ToolInput: map[string]any{"path": "services/other/main.go"}}

	envPath := filepath.Join(worktree, "services", "payments", "main.go")
	t.Setenv("CLAUDE_TOOL_INPUT_FILE_PATH", envPath)

	resp, err := GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "allow" {
		t.Fatalf("expected allow for env path, got %s", resp.Decision)
	}
}

func TestGuardrailsHookBashPolicy(t *testing.T) {
	worktree := t.TempDir()
	identity := AgentIdentity{
		Role:     "builder",
		Scope:    "services/payments",
		Worktree: worktree,
	}
	in := ClaudeHookInput{ToolName: "Bash", ToolInput: map[string]any{"command": "go test ./..."}}

	t.Setenv("CLAUDE_TOOL_INPUT_COMMAND", "")

	resp, err := GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "allow" {
		t.Fatalf("expected allow for go command, got %s", resp.Decision)
	}

	in.ToolInput = map[string]any{"command": "rm -rf /"}
	resp, err = GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "deny" {
		t.Fatalf("expected deny for unsafe command, got %s", resp.Decision)
	}

	identity.Role = "monitor"
	resp, err = GuardrailsHook(in, identity)
	if err != nil {
		t.Fatalf("guardrails: %v", err)
	}
	if resp.Decision != "deny" {
		t.Fatalf("expected deny for monitor bash, got %s", resp.Decision)
	}
}
