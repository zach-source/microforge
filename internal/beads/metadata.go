package beads

import (
	"bufio"
	"strings"
)

type Meta struct {
	Cell       string
	Role       string
	Scope      string
	Outbox     string
	Inbox      string
	Promise    string
	TurnID     string
	Worktree   string
	Kind       string
	Title      string
	ShortID    string
	SourceRole string
	Severity   string
	Conflict   bool
	Class      string
	AgentID    string
	RoleID     string
	MailboxID  string
	HookID     string
	ConvoyID   string
}

func ParseMeta(desc string) Meta {
	meta := Meta{}
	scanner := bufio.NewScanner(strings.NewReader(desc))
	inBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			if !inBlock {
				inBlock = true
				continue
			}
			break
		}
		if !inBlock {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "cell":
			meta.Cell = val
		case "role":
			meta.Role = val
		case "scope":
			meta.Scope = val
		case "outbox":
			meta.Outbox = val
		case "inbox":
			meta.Inbox = val
		case "promise":
			meta.Promise = val
		case "turn_id":
			meta.TurnID = val
		case "worktree":
			meta.Worktree = val
		case "kind":
			meta.Kind = val
		case "title":
			meta.Title = val
		case "short_id":
			meta.ShortID = val
		case "source_role":
			meta.SourceRole = val
		case "severity":
			meta.Severity = val
		case "class":
			meta.Class = val
		case "agent_id":
			meta.AgentID = val
		case "role_id":
			meta.RoleID = val
		case "mailbox_id":
			meta.MailboxID = val
		case "hook_id":
			meta.HookID = val
		case "convoy_id":
			meta.ConvoyID = val
		case "conflict":
			if strings.EqualFold(val, "true") || strings.EqualFold(val, "yes") || val == "1" {
				meta.Conflict = true
			}
		}
	}
	return meta
}

func RenderMeta(meta Meta) string {
	lines := []string{"---"}
	if meta.Cell != "" {
		lines = append(lines, "cell: "+meta.Cell)
	}
	if meta.Role != "" {
		lines = append(lines, "role: "+meta.Role)
	}
	if meta.Scope != "" {
		lines = append(lines, "scope: "+meta.Scope)
	}
	if meta.Inbox != "" {
		lines = append(lines, "inbox: "+meta.Inbox)
	}
	if meta.Outbox != "" {
		lines = append(lines, "outbox: "+meta.Outbox)
	}
	if meta.Promise != "" {
		lines = append(lines, "promise: "+meta.Promise)
	}
	if meta.TurnID != "" {
		lines = append(lines, "turn_id: "+meta.TurnID)
	}
	if meta.Worktree != "" {
		lines = append(lines, "worktree: "+meta.Worktree)
	}
	if meta.Kind != "" {
		lines = append(lines, "kind: "+meta.Kind)
	}
	if meta.Title != "" {
		lines = append(lines, "title: "+meta.Title)
	}
	if meta.ShortID != "" {
		lines = append(lines, "short_id: "+meta.ShortID)
	}
	if meta.SourceRole != "" {
		lines = append(lines, "source_role: "+meta.SourceRole)
	}
	if meta.Severity != "" {
		lines = append(lines, "severity: "+meta.Severity)
	}
	if meta.Class != "" {
		lines = append(lines, "class: "+meta.Class)
	}
	if meta.AgentID != "" {
		lines = append(lines, "agent_id: "+meta.AgentID)
	}
	if meta.RoleID != "" {
		lines = append(lines, "role_id: "+meta.RoleID)
	}
	if meta.MailboxID != "" {
		lines = append(lines, "mailbox_id: "+meta.MailboxID)
	}
	if meta.HookID != "" {
		lines = append(lines, "hook_id: "+meta.HookID)
	}
	if meta.ConvoyID != "" {
		lines = append(lines, "convoy_id: "+meta.ConvoyID)
	}
	if meta.Conflict {
		lines = append(lines, "conflict: true")
	}
	lines = append(lines, "---")
	return strings.Join(lines, "\n")
}

func StripMeta(desc string) string {
	lines := strings.Split(desc, "\n")
	if len(lines) == 0 {
		return desc
	}
	if strings.TrimSpace(lines[0]) != "---" {
		return desc
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 || end+1 >= len(lines) {
		return ""
	}
	return strings.Join(lines[end+1:], "\n")
}
