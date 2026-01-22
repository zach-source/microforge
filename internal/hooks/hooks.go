package hooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

type ClaudeHookInput struct {
	HookEventName string `json:"hook_event_name"`
	Cwd           string `json:"cwd"`
	ToolName      string `json:"tool_name,omitempty"`
	ToolInput     any    `json:"tool_input,omitempty"`
}

type StopHookResponse struct {
	Continue bool   `json:"continue"`
	Reason   string `json:"reason,omitempty"`
}

type DecisionResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

type AgentIdentity struct {
	RigName  string `json:"rig_name"`
	DBPath   string `json:"db_path"`
	CellName string `json:"cell_name"`
	Role     string `json:"role"`
	AgentID  string `json:"agent_id"`
	Scope    string `json:"scope"`
	Worktree string `json:"worktree_path"`
	Inbox   string `json:"inbox"`
	Outbox  string `json:"outbox"`
	Archive string `json:"archive"`
}

func LoadIdentityFromCWD(cwd string) (AgentIdentity, error) {
	p := filepath.Join(cwd, ".mf", "active-agent.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return AgentIdentity{}, fmt.Errorf("missing %s: %w", p, err)
	}
	var id AgentIdentity
	if err := json.Unmarshal(b, &id); err != nil { return AgentIdentity{}, err }
	return id, nil
}

func StopHook(ctx context.Context, db *sql.DB, identity AgentIdentity) (StopHookResponse, error) {
	assn, ok, err := store.ClaimNextAssignmentForAgent(db, identity.AgentID)
	if err != nil { return StopHookResponse{}, err }
	if !ok {
		return StopHookResponse{Continue: false}, nil
	}

	var title, body, kind string
	var scope sql.NullString
	row := db.QueryRow("SELECT title, body, kind, scope_prefix FROM tasks WHERE id = ?", assn.TaskID)
	if err := row.Scan(&title, &body, &kind, &scope); err != nil { return StopHookResponse{}, err }

	inboxAbs := filepath.Join(identity.Worktree, assn.InboxRel)
	mail := renderMail(identity, assn.TaskID, kind, title, body, assn.OutboxRel, assn.CompletionPromise)
	if err := util.AtomicWriteFile(inboxAbs, []byte(mail), 0o644); err != nil { return StopHookResponse{}, err }
	_ = store.MarkTaskAssigned(db, assn.TaskID)

	reason := fmt.Sprintf("New assignment claimed: %s (%s). Read %s, write %s, include promise %q when complete.",
		assn.TaskID, identity.Role, assn.InboxRel, assn.OutboxRel, assn.CompletionPromise)

	return StopHookResponse{
		Continue: true,
		Reason: reason + "\n\n=== BEGIN ASSIGNMENT ===\n" + mail + "\n=== END ASSIGNMENT ===",
	}, nil

	_ = scope
	_ = ctx
}

func renderMail(id AgentIdentity, taskID, kind, title, body, outRel, promise string) string {
	b := strings.TrimSpace(body)
	if b == "" { b = "_(no additional body provided)_" }
	scope := id.Scope
	return fmt.Sprintf(`---
task_id: %s
kind: %s
role: %s
scope: %s
out_file: %s
completion_promise: %s
---

# Goal
%s

# Details
%s

# Deliverables
1) Make changes ONLY within scope: %s
2) Write a concise report + verification steps to **%s**.
3) When complete, include the exact promise token in the outbox report: **%s**

# Workflow rules
- Builder writes code; Reviewer/Monitor do not write code.
- Run tests relevant to this scope.
- If blocked by missing info, write assumptions in the outbox report.
`, taskID, kind, id.Role, scope, outRel, promise, title, b, scope, outRel, promise)
}

func GuardrailsHook(in ClaudeHookInput, identity AgentIdentity) (DecisionResponse, error) {
	tool := strings.TrimSpace(in.ToolName)
	if tool == "Write" || tool == "Edit" {
		if identity.Role == "reviewer" || identity.Role == "monitor" {
			return DecisionResponse{Decision: "deny", Reason: "Role is read-only: " + identity.Role}, nil
		}
		fp := strings.TrimSpace(os.Getenv("CLAUDE_TOOL_INPUT_FILE_PATH"))
		if fp == "" {
			b, _ := json.Marshal(in.ToolInput)
			fp = string(b)
		}
		scope := filepath.Clean(identity.Scope)
		if scope != "." && scope != "" {
			normalized := strings.ReplaceAll(fp, "\\", "/")
			if !strings.Contains(normalized, strings.ReplaceAll(scope, "\\", "/")) {
				return DecisionResponse{Decision: "deny", Reason: fmt.Sprintf("Write/Edit outside scope %q is blocked (saw: %s)", identity.Scope, fp)}, nil
			}
		}
	}
	return DecisionResponse{Decision: "allow"}, nil
}
