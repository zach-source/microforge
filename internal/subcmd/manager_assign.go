package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func ManagerAssign(home, rigName string, rest []string) error {
	role := "builder"
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--role" && i+1 < len(rest) {
			role = rest[i+1]
			i++
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	warnContextMismatch(home, rigName, "manager assign")
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return err
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	turnID := ""
	if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
		turnID = strings.TrimSpace(state.ID)
	}
	assigned := 0
	for _, issue := range issues {
		typeLower := strings.ToLower(issue.Type)
		if typeLower != "task" && typeLower != "request" && typeLower != "observation" {
			continue
		}
		if issue.Status == "closed" || issue.Status == "done" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		cell := matchCellByScope(cells, meta.Scope)
		if cell == nil {
			continue
		}
		if err := ensureCellBootstrapped(home, rigName, cell.Name, role, true); err != nil {
			return err
		}
		if err := beadLimit(home, rigName, cell.Name, turnID); err != nil {
			return err
		}
		inboxRel := fmt.Sprintf("mail/inbox/%s.md", issue.ID)
		outboxRel := fmt.Sprintf("mail/outbox/%s.md", issue.ID)
		assnMeta := beads.Meta{
			Cell:     cell.Name,
			Role:     role,
			Scope:    cell.ScopePrefix,
			Inbox:    inboxRel,
			Outbox:   outboxRel,
			Promise:  "DONE",
			TurnID:   turnID,
			Worktree: cell.WorktreePath,
		}
		assn, err := client.Create(nil, beads.CreateRequest{
			Title:       "Assignment " + issue.ID,
			Type:        "assignment",
			Priority:    issue.Priority,
			Status:      "open",
			Description: beads.RenderMeta(assnMeta) + "\n\n" + issue.Title,
			Deps:        []string{"related:" + issue.ID},
		})
		if err != nil {
			return err
		}
		mail, _ := writeAssignmentInbox(cell.WorktreePath, inboxRel, outboxRel, "DONE", issue)
		_ = createMailBead(client, assnMeta, "Mail "+assn.ID, mail, []string{"related:" + assn.ID})
		_, _ = client.UpdateStatus(nil, issue.ID, "in_progress")
		emitOrchestrationEvent(cfg.RepoPath, beads.Meta{
			Cell:   cell.Name,
			Role:   role,
			Scope:  cell.ScopePrefix,
			TurnID: turnID,
			Kind:   "assignment_created",
		}, fmt.Sprintf("Assignment %s created", assn.ID), []string{"related:" + issue.ID})
		assigned++
	}
	fmt.Printf("Auto-assigned %d bead(s)\n", assigned)
	return nil
}
