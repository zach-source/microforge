package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Epic(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge epic <create|add-task|assign|status|close|conflict|design|tree> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "create":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic create --title <t> [--body <md>] [--short-id <id>]")
		}
		rigName := rest[0]
		var title, body, shortID string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--title":
				if i+1 < len(rest) {
					title = rest[i+1]
					i++
				}
			case "--body":
				if i+1 < len(rest) {
					body = rest[i+1]
					i++
				}
			case "--short-id":
				if i+1 < len(rest) {
					shortID = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(title) == "" {
			return fmt.Errorf("--title is required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		meta := beads.Meta{Title: title, Kind: "epic", ShortID: strings.TrimSpace(shortID)}
		desc := beads.RenderMeta(meta)
		if strings.TrimSpace(body) != "" {
			desc += "\n\n" + strings.TrimSpace(body)
		}
		issue, err := client.Create(nil, beads.CreateRequest{
			Title:       title,
			Type:        "epic",
			Priority:    "p2",
			Status:      "open",
			Description: desc,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Created epic %s\n", issue.ID)
		return nil

	case "add-task":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic add-task --epic <id> --task <id>")
		}
		rigName := rest[0]
		var epicID, taskID string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" || strings.TrimSpace(taskID) == "" {
			return fmt.Errorf("--epic and --task are required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		if err := client.DepAdd(nil, taskID, "related:"+epicID); err != nil {
			return err
		}
		fmt.Printf("Added task %s to epic %s\n", taskID, epicID)
		return nil

	case "status":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic status --epic <id>")
		}
		rigName := rest[0]
		var epicID string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" {
			return fmt.Errorf("--epic is required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issues, err := client.List(nil)
		if err != nil {
			return err
		}
		counts := map[string]int{}
		for _, issue := range issues {
			if strings.ToLower(issue.Type) != "task" {
				continue
			}
			if !hasDep(issue, epicID) {
				continue
			}
			counts[issue.Status]++
		}
		for status, count := range counts {
			fmt.Printf("%s\t%d\n", status, count)
		}
		return nil

	case "assign":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic assign --epic <id> [--role <role>]")
		}
		rigName := rest[0]
		var epicID, role string
		role = "builder"
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			case "--role":
				if i+1 < len(rest) {
					role = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" {
			return fmt.Errorf("--epic is required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
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
			if strings.ToLower(issue.Type) != "task" {
				continue
			}
			if issue.Status == "closed" || issue.Status == "done" {
				continue
			}
			if !hasDep(issue, epicID) {
				continue
			}
			meta := beads.ParseMeta(issue.Description)
			cell := matchCellByScope(cells, meta.Scope)
			if cell == nil {
				continue
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
			assnDesc := beads.RenderMeta(assnMeta) + "\n\n" + issue.Title
			if _, err := client.Create(nil, beads.CreateRequest{
				Title:       "Assignment " + issue.ID,
				Type:        "assignment",
				Priority:    issue.Priority,
				Status:      "open",
				Description: assnDesc,
				Deps:        []string{"related:" + issue.ID},
			}); err != nil {
				return err
			}
			assigned++
		}
		fmt.Printf("Assigned %d task(s)\n", assigned)
		return nil

	case "close":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic close --epic <id>")
		}
		rigName := rest[0]
		var epicID string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" {
			return fmt.Errorf("--epic is required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issues, err := client.List(nil)
		if err != nil {
			return err
		}
		hasReview := false
		for _, issue := range issues {
			if strings.ToLower(issue.Type) != "task" {
				continue
			}
			if !hasDep(issue, epicID) {
				continue
			}
			if issue.Status != "closed" && issue.Status != "done" {
				return fmt.Errorf("epic has incomplete tasks")
			}
			meta := beads.ParseMeta(issue.Description)
			if meta.Kind == "review" {
				hasReview = true
			}
		}
		if !hasReview {
			return fmt.Errorf("epic close requires a completed review task")
		}
		_, err = client.Close(nil, epicID, "epic closed")
		if err != nil {
			return err
		}
		fmt.Printf("Closed epic %s\n", epicID)
		return nil

	case "conflict":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic conflict --epic <id> --cell <cell> --details <text>")
		}
		rigName := rest[0]
		var epicID, cellName, details string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			case "--cell":
				if i+1 < len(rest) {
					cellName = rest[i+1]
					i++
				}
			case "--details":
				if i+1 < len(rest) {
					details = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" || strings.TrimSpace(cellName) == "" {
			return fmt.Errorf("--epic and --cell are required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		payload := fmt.Sprintf("{\"epic\":%q,\"details\":%q}", epicID, details)
		meta := beads.Meta{Cell: cellName, SourceRole: "reviewer"}
		_, err = client.Create(nil, beads.CreateRequest{
			Title:       "Conflict for epic " + epicID,
			Type:        "request",
			Priority:    "p1",
			Status:      "open",
			Description: beads.RenderMeta(meta) + "\n\n" + payload,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Created conflict request for epic %s\n", epicID)
		return nil

	case "design":
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge epic design <id|short-id>")
		}
		rigName := rest[0]
		key := rest[1]
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issues, err := client.List(nil)
		if err != nil {
			return err
		}
		epic, _ := findEpicByKey(issues, key)
		if epic.ID == "" {
			return fmt.Errorf("epic not found: %s", key)
		}
		cells, err := rig.ListCellConfigs(home, rigName)
		if err != nil {
			return err
		}
		created := 0
		for _, cell := range cells {
			title := fmt.Sprintf("Plan epic %s for %s", epic.ID, cell.Name)
			meta := beads.Meta{Scope: cell.ScopePrefix, Kind: "plan", Title: title}
			desc := beads.RenderMeta(meta)
			task, err := client.Create(nil, beads.CreateRequest{
				Title:       title,
				Type:        "task",
				Priority:    "p2",
				Status:      "open",
				Description: desc,
				Deps:        []string{"related:" + epic.ID},
			})
			if err != nil {
				return err
			}
			role := preferredRoleForCell(cell.WorktreePath)
			if err := Assign(home, []string{rigName, "--task", task.ID, "--cell", cell.Name, "--role", role}); err != nil {
				return err
			}
			_ = Agent(home, []string{"wake", rigName, cell.Name, role})
			created++
		}
		fmt.Printf("Epic design queued %d task(s) for %s\n", created, epic.ID)
		return nil

	case "tree":
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge epic tree <id|short-id>")
		}
		rigName := rest[0]
		key := rest[1]
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issues, err := client.List(nil)
		if err != nil {
			return err
		}
		epic, meta := findEpicByKey(issues, key)
		if epic.ID == "" {
			return fmt.Errorf("epic not found: %s", key)
		}
		label := epic.ID
		if strings.TrimSpace(meta.ShortID) != "" {
			label = fmt.Sprintf("%s (%s)", epic.ID, meta.ShortID)
		}
		fmt.Printf("Epic %s: %s [%s]\n", label, epic.Title, epic.Status)
		for _, issue := range issues {
			if strings.ToLower(issue.Type) != "task" {
				continue
			}
			if !hasDep(issue, epic.ID) {
				continue
			}
			assn := findAssignmentForTask(issues, issue.ID)
			assnStatus := "-"
			if assn.ID != "" {
				assnStatus = assn.Status
			}
			fmt.Printf("  - %s [%s] (assignment=%s)\n", issue.ID, issue.Status, assnStatus)
		}
		return nil

	default:
		return fmt.Errorf("unknown epic subcommand: %s", op)
	}
}

func findEpicByKey(issues []beads.Issue, key string) (beads.Issue, beads.Meta) {
	key = strings.TrimSpace(key)
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "epic" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if issue.ID == key || strings.EqualFold(meta.ShortID, key) {
			return issue, meta
		}
	}
	return beads.Issue{}, beads.Meta{}
}

func preferredRoleForCell(worktree string) string {
	if strings.TrimSpace(worktree) == "" {
		return "builder"
	}
	if _, err := os.Stat(filepath.Join(worktree, ".mf", "active-agent-cell.json")); err == nil {
		return "cell"
	}
	return "builder"
}

func findAssignmentForTask(issues []beads.Issue, taskID string) beads.Issue {
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "assignment" {
			continue
		}
		for _, dep := range issue.Deps {
			if dep == "related:"+taskID {
				return issue
			}
		}
	}
	return beads.Issue{}
}

func hasDep(issue beads.Issue, id string) bool {
	for _, dep := range issue.Deps {
		if strings.Contains(dep, id) {
			return true
		}
	}
	return false
}

func matchCellByScope(cells []rig.CellConfig, scope string) *rig.CellConfig {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil
	}
	var best *rig.CellConfig
	for i := range cells {
		c := cells[i]
		if strings.HasPrefix(scope, c.ScopePrefix) {
			if best == nil || len(c.ScopePrefix) > len(best.ScopePrefix) {
				best = &c
			}
		}
	}
	return best
}
