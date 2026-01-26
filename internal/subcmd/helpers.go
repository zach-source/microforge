package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/context"
	"github.com/example/microforge/internal/util"
)

func defaultIfEmpty(val, def string) string {
	if strings.TrimSpace(val) == "" {
		return def
	}
	return val
}

func writeAssignmentInbox(worktree, inboxRel, outboxRel, promise string, issue beads.Issue) (string, error) {
	if strings.TrimSpace(worktree) == "" {
		return "", nil
	}
	if strings.TrimSpace(inboxRel) == "" {
		inboxRel = filepath.Join("mail", "inbox", fmt.Sprintf("%s.md", issue.ID))
	}
	if strings.TrimSpace(outboxRel) == "" {
		outboxRel = filepath.Join("mail", "outbox", fmt.Sprintf("%s.md", issue.ID))
	}
	body := strings.TrimSpace(beads.StripMeta(issue.Description))
	if body == "" {
		body = "_(no additional body provided)_"
	}
	mail := fmt.Sprintf(`---
task_id: %s
kind: %s
out_file: %s
completion_promise: %s
---

# Goal
%s

# Details
%s
`, issue.ID, issue.Type, outboxRel, promise, issue.Title, body)
	inboxAbs := filepath.Join(worktree, inboxRel)
	if err := util.EnsureDir(filepath.Dir(inboxAbs)); err != nil {
		return mail, err
	}
	return mail, util.AtomicWriteFile(inboxAbs, []byte(mail), 0o644)
}

func ensureAgentBead(repo string, spec AgentSpec) (string, error) {
	client := beads.Client{RepoPath: repo}
	issues, err := client.List(nil)
	if err != nil {
		return "", err
	}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "agent" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.EqualFold(meta.Cell, spec.Name) {
			return issue.ID, nil
		}
	}
	meta := beads.Meta{
		Cell:  spec.Name,
		Scope: spec.Scope,
		Kind:  "agent",
		Title: spec.Name,
		Class: spec.Class,
	}
	desc := beads.RenderMeta(meta)
	if strings.TrimSpace(spec.Description) != "" {
		desc += "\n\n" + strings.TrimSpace(spec.Description)
	}
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       "Agent " + spec.Name,
		Type:        "agent",
		Priority:    "p2",
		Status:      "open",
		Description: desc,
	})
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

func ensureRoleBead(issues []beads.Issue, client beads.Client, cell, role, scope, guide string) (beads.Issue, []beads.Issue, error) {
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "role" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.EqualFold(meta.Cell, cell) && strings.EqualFold(meta.Role, role) {
			return issue, issues, nil
		}
	}
	meta := beads.Meta{Cell: cell, Role: role, Scope: scope, Kind: "role", Title: role}
	desc := beads.RenderMeta(meta)
	if strings.TrimSpace(guide) != "" {
		desc += "\n\n" + strings.TrimSpace(guide)
	}
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       fmt.Sprintf("Role %s/%s", cell, role),
		Type:        "role",
		Priority:    "p2",
		Status:      "open",
		Description: desc,
	})
	if err != nil {
		return beads.Issue{}, issues, err
	}
	return issue, append(issues, issue), nil
}

func ensureMailboxBead(issues []beads.Issue, client beads.Client, cell, role, inboxRel, outboxRel, worktree string) (beads.Issue, []beads.Issue, error) {
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "mailbox" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.EqualFold(meta.Cell, cell) && strings.EqualFold(meta.Role, role) {
			return issue, issues, nil
		}
	}
	meta := beads.Meta{
		Cell:     cell,
		Role:     role,
		Inbox:    inboxRel,
		Outbox:   outboxRel,
		Worktree: worktree,
		Kind:     "mailbox",
		Title:    fmt.Sprintf("%s/%s mailbox", cell, role),
	}
	desc := beads.RenderMeta(meta)
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       fmt.Sprintf("Mailbox %s/%s", cell, role),
		Type:        "mailbox",
		Priority:    "p3",
		Status:      "open",
		Description: desc,
	})
	if err != nil {
		return beads.Issue{}, issues, err
	}
	return issue, append(issues, issue), nil
}

func ensureHookBead(issues []beads.Issue, client beads.Client, cell, role, scope string) (beads.Issue, []beads.Issue, error) {
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "hook" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.EqualFold(meta.Cell, cell) && strings.EqualFold(meta.Role, role) {
			return issue, issues, nil
		}
	}
	meta := beads.Meta{Cell: cell, Role: role, Scope: scope, Kind: "hook"}
	desc := beads.RenderMeta(meta)
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       fmt.Sprintf("Hook %s/%s", cell, role),
		Type:        "hook",
		Priority:    "p3",
		Status:      "open",
		Description: desc,
	})
	if err != nil {
		return beads.Issue{}, issues, err
	}
	return issue, append(issues, issue), nil
}

func createMailBead(client beads.Client, meta beads.Meta, title, body string, deps []string) error {
	meta.Kind = "mail"
	desc := beads.RenderMeta(meta) + "\n\n" + strings.TrimSpace(body)
	_, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "mail",
		Priority:    "p3",
		Status:      "open",
		Description: desc,
		Deps:        deps,
	})
	return err
}

func emitOrchestrationEvent(repo string, meta beads.Meta, title string, deps []string) {
	if strings.TrimSpace(repo) == "" {
		return
	}
	client := beads.Client{RepoPath: repo}
	if strings.TrimSpace(meta.Kind) == "" {
		meta.Kind = "orchestration"
	}
	desc := beads.RenderMeta(meta)
	_, _ = client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "event",
		Priority:    "p3",
		Status:      "open",
		Description: desc,
		Deps:        deps,
	})
}

func ensureBeadsTypes(repo string) error {
	required := []string{
		"assignment", "plan", "improve", "fix", "review", "monitor", "doc",
		"turn", "epic", "event", "request", "observation", "decision",
		"contract", "pr", "build", "deploy", "task",
		"agent", "role", "mailbox", "hook", "convoy", "mail",
	}
	configPath := filepath.Join(repo, ".beads", "config.yaml")
	b, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			line := fmt.Sprintf("types.custom: %q\n", strings.Join(required, ","))
			return util.AtomicWriteFile(configPath, []byte(line), 0o644)
		}
		return err
	}
	lines := strings.Split(string(b), "\n")
	found := false
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "types.custom:") {
			continue
		}
		found = true
		existing := strings.TrimSpace(strings.TrimPrefix(line, "types.custom:"))
		existing = strings.Trim(existing, "\"")
		items := splitCSV(existing)
		set := map[string]bool{}
		for _, item := range items {
			set[item] = true
		}
		for _, req := range required {
			if !set[req] {
				items = append(items, req)
				set[req] = true
			}
		}
		sort.Strings(items)
		lines[i] = fmt.Sprintf("types.custom: %q", strings.Join(items, ","))
	}
	if !found {
		lines = append(lines, fmt.Sprintf("types.custom: %q", strings.Join(required, ",")))
	}
	out := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	return util.AtomicWriteFile(configPath, []byte(out), 0o644)
}

func ensureClaudeSymlink(repo, worktree string) {
	if strings.TrimSpace(repo) == "" || strings.TrimSpace(worktree) == "" {
		return
	}
	src := filepath.Join(repo, "CLAUDE.md")
	if _, err := os.Stat(src); err != nil {
		return
	}
	dst := filepath.Join(worktree, "CLAUDE.md")
	if _, err := os.Lstat(dst); err == nil {
		return
	}
	if err := os.Symlink(src, dst); err == nil {
		return
	}
	if b, err := os.ReadFile(src); err == nil {
		_ = util.AtomicWriteFile(dst, b, 0o644)
	}
}

func warnContextMismatch(home, rigName, action string) {
	state, err := context.Load(home)
	if err != nil {
		return
	}
	active := strings.TrimSpace(state.ActiveRig)
	if active == "" || rigName == "" || active == rigName {
		return
	}
	if action == "" {
		action = "command"
	}
	fmt.Printf("Warning: %s is using rig %q but active rig is %q\n", action, rigName, active)
}
