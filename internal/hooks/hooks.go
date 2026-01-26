package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
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
	RigName     string `json:"rig_name"`
	RigHome     string `json:"rig_home"`
	RepoPath    string `json:"repo_path"`
	CellName    string `json:"cell_name"`
	Role        string `json:"role"`
	Scope       string `json:"scope"`
	Worktree    string `json:"worktree_path"`
	TmuxSession string `json:"tmux_session"`
	Inbox       string `json:"inbox"`
	Outbox      string `json:"outbox"`
	Archive     string `json:"archive"`
	AgentID     string `json:"agent_id,omitempty"`
	RoleID      string `json:"role_id,omitempty"`
	MailboxID   string `json:"mailbox_id,omitempty"`
	HookID      string `json:"hook_id,omitempty"`
	Class       string `json:"class,omitempty"`
}

func LoadIdentityFromCWD(cwd string) (AgentIdentity, error) {
	p := filepath.Join(cwd, ".mf", "active-agent.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return AgentIdentity{}, fmt.Errorf("missing %s: %w", p, err)
	}
	var id AgentIdentity
	if err := json.Unmarshal(b, &id); err != nil {
		return AgentIdentity{}, err
	}
	return id, nil
}

func StopHook(ctx context.Context, client beads.Client, identity AgentIdentity) (StopHookResponse, error) {
	ready, err := client.Ready(ctx)
	if err != nil {
		return StopHookResponse{}, err
	}
	turnID := currentTurnID(identity)
	var chosen beads.Issue
	var meta beads.Meta
	for _, issue := range ready {
		if strings.ToLower(issue.Type) != "assignment" {
			continue
		}
		m := beads.ParseMeta(issue.Description)
		if m.Cell != "" && m.Cell != identity.CellName {
			continue
		}
		if m.Role != "" && m.Role != identity.Role {
			continue
		}
		if turnID != "" && m.TurnID != "" && m.TurnID != turnID {
			continue
		}
		chosen = issue
		meta = m
		break
	}
	if chosen.ID == "" {
		UpdateHeartbeat(identity, "idle", "", turnID, "")
		updateHookIdle(ctx, client, identity, turnID)
		emitHookIdleEvent(ctx, client, identity, turnID)
		return StopHookResponse{Continue: false}, nil
	}
	_, _ = client.UpdateStatus(ctx, chosen.ID, "in_progress")

	inboxRel := meta.Inbox
	if strings.TrimSpace(inboxRel) == "" {
		inboxRel = filepath.Join("mail", "inbox", fmt.Sprintf("%s.md", chosen.ID))
	}
	outboxRel := meta.Outbox
	if strings.TrimSpace(outboxRel) == "" {
		outboxRel = filepath.Join("mail", "outbox", fmt.Sprintf("%s.md", chosen.ID))
	}
	promise := meta.Promise
	if strings.TrimSpace(promise) == "" {
		promise = "DONE:" + chosen.ID
	}

	body := strings.TrimSpace(beads.StripMeta(chosen.Description))
	inboxAbs := filepath.Join(identity.Worktree, inboxRel)
	mail := renderMail(identity, chosen.ID, chosen.Type, chosen.Title, body, outboxRel, promise)
	if err := util.AtomicWriteFile(inboxAbs, []byte(mail), 0o644); err != nil {
		return StopHookResponse{}, err
	}

	updateHookBead(ctx, client, identity, chosen, meta, mail)
	emitHookEvent(ctx, client, identity, chosen, meta)

	reason := fmt.Sprintf("New assignment claimed: %s (%s). Read %s, write %s, include promise %q when complete.",
		chosen.ID, identity.Role, inboxRel, outboxRel, promise)
	if shouldResetContext() {
		reason = "CONTEXT RESET REQUIRED: Drop prior task context. Start fresh with this assignment only.\n" + reason
	}

	UpdateHeartbeat(identity, "claimed", chosen.ID, turnID, "")
	return StopHookResponse{
		Continue: true,
		Reason:   reason + "\n\n=== BEGIN ASSIGNMENT ===\n" + mail + "\n=== END ASSIGNMENT ===",
	}, nil
}

func shouldResetContext() bool {
	val := strings.TrimSpace(os.Getenv("MF_CLEAR_CONTEXT_ON_CLAIM"))
	if val == "" {
		return true
	}
	return strings.EqualFold(val, "1") || strings.EqualFold(val, "true") || strings.EqualFold(val, "yes")
}

func updateHookBead(ctx context.Context, client beads.Client, identity AgentIdentity, issue beads.Issue, meta beads.Meta, mail string) {
	if strings.TrimSpace(identity.HookID) == "" {
		return
	}
	descMeta := beads.Meta{
		Cell:    identity.CellName,
		Role:    identity.Role,
		Scope:   identity.Scope,
		Inbox:   meta.Inbox,
		Outbox:  meta.Outbox,
		Promise: meta.Promise,
		TurnID:  meta.TurnID,
		Kind:    "hook",
	}
	desc := beads.RenderMeta(descMeta) + "\n\n" + mail
	if _, err := client.UpdateDescription(ctx, identity.HookID, desc); err != nil {
		if err == beads.ErrUpdateDescriptionUnsupported {
			_, _ = client.Create(ctx, beads.CreateRequest{
				Title:       fmt.Sprintf("Hook event %s/%s", identity.CellName, identity.Role),
				Type:        "hook",
				Priority:    "p2",
				Status:      "open",
				Description: desc,
				Deps:        []string{"related:" + issue.ID},
			})
		}
	}
}

func emitHookEvent(ctx context.Context, client beads.Client, identity AgentIdentity, issue beads.Issue, meta beads.Meta) {
	descMeta := beads.Meta{
		Cell:    identity.CellName,
		Role:    identity.Role,
		Scope:   identity.Scope,
		TurnID:  meta.TurnID,
		Kind:    "hook_claim",
		Title:   issue.Title,
		AgentID: identity.AgentID,
		HookID:  identity.HookID,
	}
	desc := beads.RenderMeta(descMeta)
	_, _ = client.Create(ctx, beads.CreateRequest{
		Title:       fmt.Sprintf("Hook claimed %s", issue.ID),
		Type:        "event",
		Priority:    "p3",
		Status:      "open",
		Description: desc,
		Deps:        []string{"related:" + issue.ID},
	})
}

func updateHookIdle(ctx context.Context, client beads.Client, identity AgentIdentity, turnID string) {
	if strings.TrimSpace(identity.HookID) == "" {
		return
	}
	descMeta := beads.Meta{
		Cell:   identity.CellName,
		Role:   identity.Role,
		Scope:  identity.Scope,
		TurnID: turnID,
		Kind:   "hook",
	}
	desc := beads.RenderMeta(descMeta) + "\n\nIDLE"
	if _, err := client.UpdateDescription(ctx, identity.HookID, desc); err != nil {
		if err == beads.ErrUpdateDescriptionUnsupported {
			_, _ = client.Create(ctx, beads.CreateRequest{
				Title:       fmt.Sprintf("Hook idle %s/%s", identity.CellName, identity.Role),
				Type:        "hook",
				Priority:    "p3",
				Status:      "open",
				Description: desc,
			})
		}
	}
}

func emitHookIdleEvent(ctx context.Context, client beads.Client, identity AgentIdentity, turnID string) {
	descMeta := beads.Meta{
		Cell:    identity.CellName,
		Role:    identity.Role,
		Scope:   identity.Scope,
		TurnID:  turnID,
		Kind:    "hook_idle",
		AgentID: identity.AgentID,
		HookID:  identity.HookID,
	}
	desc := beads.RenderMeta(descMeta)
	_, _ = client.Create(ctx, beads.CreateRequest{
		Title:       fmt.Sprintf("Hook idle %s/%s", identity.CellName, identity.Role),
		Type:        "event",
		Priority:    "p3",
		Status:      "open",
		Description: desc,
	})
}

func currentTurnID(identity AgentIdentity) string {
	home := strings.TrimSpace(identity.RigHome)
	if home == "" {
		home = rig.DefaultHome()
		if v := strings.TrimSpace(os.Getenv("MF_HOME")); v != "" {
			home = v
		}
	}
	statePath := rig.TurnStatePath(home, identity.RigName)
	state, err := turn.Load(statePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(state.ID)
}

func renderMail(id AgentIdentity, taskID, kind, title, body, outRel, promise string) string {
	b := strings.TrimSpace(body)
	if b == "" {
		b = "_(no additional body provided)_"
	}
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
		if identity.Role == "reviewer" || identity.Role == "monitor" || identity.Role == "architect" {
			return DecisionResponse{Decision: "deny", Reason: "Role is read-only: " + identity.Role}, nil
		}
		fp := strings.TrimSpace(extractToolPath(in))
		if fp != "" && !pathWithinScope(identity, fp) {
			return DecisionResponse{Decision: "deny", Reason: fmt.Sprintf("Write/Edit outside scope %q is blocked (saw: %s)", identity.Scope, fp)}, nil
		}
	}
	if tool == "Bash" || tool == "PermissionRequest" {
		if identity.Role == "reviewer" || identity.Role == "monitor" || identity.Role == "architect" {
			return DecisionResponse{Decision: "deny", Reason: "Role is read-only: " + identity.Role}, nil
		}
		if identity.Role == "builder" {
			cmd := strings.TrimSpace(extractToolCommand(in))
			if cmd != "" && !isAllowedBuilderCommand(cmd) {
				return DecisionResponse{Decision: "deny", Reason: "Command not allowed for builder policy: " + cmd}, nil
			}
		}
	}
	return DecisionResponse{Decision: "allow"}, nil
}

func extractToolPath(in ClaudeHookInput) string {
	if fp := strings.TrimSpace(os.Getenv("CLAUDE_TOOL_INPUT_FILE_PATH")); fp != "" {
		return fp
	}
	switch v := in.ToolInput.(type) {
	case map[string]any:
		if p, ok := v["path"].(string); ok {
			return p
		}
		if p, ok := v["file_path"].(string); ok {
			return p
		}
	case string:
		return v
	}
	return ""
}

func extractToolCommand(in ClaudeHookInput) string {
	if cmd := strings.TrimSpace(os.Getenv("CLAUDE_TOOL_INPUT_COMMAND")); cmd != "" {
		return cmd
	}
	switch v := in.ToolInput.(type) {
	case map[string]any:
		if c, ok := v["command"].(string); ok {
			return c
		}
		if c, ok := v["cmd"].(string); ok {
			return c
		}
	case string:
		return v
	}
	return ""
}

func isAllowedBuilderCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return true
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return true
	}
	allowed := map[string]bool{
		"go":   true,
		"make": true,
		"git":  true,
		"rg":   true,
		"sed":  true,
		"cat":  true,
		"ls":   true,
		"pwd":  true,
		"env":  true,
		"grep": true,
	}
	return allowed[parts[0]]
}

func pathWithinScope(identity AgentIdentity, fp string) bool {
	scope := strings.TrimSpace(identity.Scope)
	if scope == "" || scope == "." {
		return true
	}
	base := scope
	if !filepath.IsAbs(base) {
		base = filepath.Join(identity.Worktree, base)
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return true
	}

	target := fp
	if !filepath.IsAbs(target) {
		target = filepath.Join(identity.Worktree, target)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return true
	}

	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return true
	}
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}
	prefix := ".." + string(os.PathSeparator)
	if strings.HasPrefix(rel, prefix) {
		return false
	}
	return true
}
