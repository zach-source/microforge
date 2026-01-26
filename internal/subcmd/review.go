package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Review(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge review create <rig> --title <title> --cell <cell> [--scope <path>] [--turn <id>]")
	}
	op := args[0]
	rest := args[1:]
	if op != "create" {
		return fmt.Errorf("unknown review subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge review create <rig> --title <title> --cell <cell> [--scope <path>] [--turn <id>]")
	}
	rigName := rest[0]
	var title, cell, scope, turnID string
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--title":
			if i+1 < len(rest) {
				title = rest[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(rest) {
				cell = rest[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(rest) {
				scope = rest[i+1]
				i++
			}
		case "--turn":
			if i+1 < len(rest) {
				turnID = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(title) == "" || strings.TrimSpace(cell) == "" {
		return fmt.Errorf("--title and --cell are required")
	}
	if strings.TrimSpace(turnID) == "" {
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
	}
	if err := beadLimit(home, rigName, cell, turnID); err != nil {
		return err
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	meta := beads.Meta{Cell: cell, Scope: scope, TurnID: turnID, Role: "reviewer"}
	reviewIssue, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "review",
		Priority:    "p2",
		Status:      "open",
		Description: beads.RenderMeta(meta) + "\n\n" + "Review required",
	})
	if err != nil {
		return err
	}
	cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cell))
	if err != nil {
		return err
	}
	inboxRel := fmt.Sprintf("mail/inbox/%s.md", reviewIssue.ID)
	outboxRel := fmt.Sprintf("mail/outbox/%s.md", reviewIssue.ID)
	assnMeta := beads.Meta{
		Cell:     cell,
		Role:     "reviewer",
		Scope:    scope,
		Inbox:    inboxRel,
		Outbox:   outboxRel,
		Promise:  "DONE",
		TurnID:   turnID,
		Worktree: cellCfg.WorktreePath,
	}
	assn, err := client.Create(nil, beads.CreateRequest{
		Title:       "Assignment " + reviewIssue.ID,
		Type:        "assignment",
		Priority:    "p2",
		Status:      "open",
		Description: beads.RenderMeta(assnMeta) + "\n\n" + title,
		Deps:        []string{"review:" + reviewIssue.ID},
	})
	if err != nil {
		return err
	}
	mail, _ := writeAssignmentInbox(cellCfg.WorktreePath, inboxRel, outboxRel, "DONE", reviewIssue)
	_ = createMailBead(client, assnMeta, "Mail "+assn.ID, mail, []string{"related:" + assn.ID})
	fmt.Printf("Created review %s\n", reviewIssue.ID)
	return nil
}
