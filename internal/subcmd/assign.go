package subcmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Assign(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge assign <rig> --task <id> --cell <cell> --role <role> [--promise <token>]")
	}
	rigName := args[0]
	var taskID, cellName, role, promise string
	promise = "DONE"
	quick := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--task":
			if i+1 < len(args) {
				taskID = args[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(args) {
				cellName = args[i+1]
				i++
			}
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--promise":
			if i+1 < len(args) {
				promise = args[i+1]
				i++
			}
		case "--quick":
			quick = true
		}
	}
	if strings.TrimSpace(taskID) == "" || strings.TrimSpace(cellName) == "" || strings.TrimSpace(role) == "" {
		return fmt.Errorf("--task, --cell, and --role are required")
	}
	warnContextMismatch(home, rigName, "assign")
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
	if err != nil {
		return err
	}
	if err := ensureCellBootstrapped(home, rigName, cellName, role, true); err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "assignment" {
			continue
		}
		if issue.Status == "closed" || issue.Status == "done" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if meta.Cell != "" && !strings.EqualFold(meta.Cell, cellName) {
			continue
		}
		if meta.Role != "" && !strings.EqualFold(meta.Role, role) {
			continue
		}
		for _, dep := range issue.Deps {
			if dep == "related:"+taskID {
				fmt.Printf("Assignment already exists for task %s\n", taskID)
				return nil
			}
		}
		if strings.Contains(issue.Title, taskID) {
			fmt.Printf("Assignment already exists for task %s\n", taskID)
			return nil
		}
	}

	inboxRel := filepath.Join("mail/inbox", fmt.Sprintf("%s.md", taskID))
	outboxRel := filepath.Join("mail/outbox", fmt.Sprintf("%s.md", taskID))
	turnID := ""
	if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
		turnID = strings.TrimSpace(state.ID)
	}
	if err := beadLimit(home, rigName, cellName, turnID); err != nil {
		return err
	}
	meta := beads.Meta{
		Cell:     cellName,
		Role:     role,
		Scope:    cellCfg.ScopePrefix,
		Inbox:    inboxRel,
		Outbox:   outboxRel,
		Promise:  promise,
		TurnID:   turnID,
		Worktree: cellCfg.WorktreePath,
	}
	body := "Assigned work for task " + taskID
	if task, err := client.Show(nil, taskID); err == nil {
		if strings.TrimSpace(task.Title) != "" {
			body = fmt.Sprintf("Assigned work for task %s: %s", taskID, task.Title)
		}
	}
	desc := beads.RenderMeta(meta) + "\n\n" + body
	req := beads.CreateRequest{
		Title:       "Assignment " + taskID,
		Type:        "assignment",
		Priority:    "p2",
		Status:      "open",
		Description: desc,
		Deps:        []string{"related:" + taskID},
	}
	assn, err := client.Create(nil, req)
	if err != nil {
		return err
	}
	if task, err := client.Show(nil, taskID); err == nil {
		mail, _ := writeAssignmentInbox(cellCfg.WorktreePath, inboxRel, outboxRel, promise, task)
		_ = createMailBead(client, meta, "Mail "+assn.ID, mail, []string{"related:" + assn.ID})
	}
	emitOrchestrationEvent(cfg.RepoPath, beads.Meta{
		Cell:      cellName,
		Role:      role,
		Scope:     cellCfg.ScopePrefix,
		TurnID:    turnID,
		Kind:      "assignment_created",
		MailboxID: "",
	}, fmt.Sprintf("Assignment %s created", assn.ID), []string{"related:" + taskID})
	fmt.Printf("Assigned task %s -> %s/%s (assignment %s)\n", taskID, cellName, role, assn.ID)
	if quick {
		_ = Agent(home, []string{"wake", rigName, cellName, role})
	}
	return nil
}
