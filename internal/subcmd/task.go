package subcmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
)

func Task(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge task <create|update|list|split|decompose|complete|delete> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "create":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task create <rig> --title <t> [--body <md>] [--scope <path>] [--kind <kind>]")
		}
		rigName := rest[0]
		var title, body, scope, kind string
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
			case "--scope":
				if i+1 < len(rest) {
					scope = rest[i+1]
					i++
				}
			case "--kind":
				if i+1 < len(rest) {
					kind = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(title) == "" {
			return fmt.Errorf("--title is required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return fmt.Errorf("loading rig %s: %w", rigName, err)
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		meta := beads.Meta{Scope: scope, Kind: kind, Title: title}
		desc := beads.RenderMeta(meta)
		if strings.TrimSpace(body) != "" {
			desc += "\n\n" + body
		}
		issue, err := client.Create(nil, beads.CreateRequest{
			Title:       title,
			Type:        "task",
			Priority:    "p2",
			Status:      "open",
			Description: desc,
		})
		if err != nil {
			return fmt.Errorf("creating task: %w", err)
		}
		fmt.Printf("Created task %s\n", issue.ID)
		return nil

	case "update":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task update <rig> --task <id> --scope <path-prefix>")
		}
		rigName := rest[0]
		var taskID, scope string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			case "--scope":
				if i+1 < len(rest) {
					scope = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(taskID) == "" || strings.TrimSpace(scope) == "" {
			return fmt.Errorf("--task and --scope are required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return fmt.Errorf("loading rig %s: %w", rigName, err)
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issue, err := client.Show(nil, taskID)
		if err != nil {
			return fmt.Errorf("showing task %s: %w", taskID, err)
		}
		if strings.ToLower(issue.Type) != "task" {
			return fmt.Errorf("bead %s is not a task (type=%s)", issue.ID, issue.Type)
		}
		meta := beads.ParseMeta(issue.Description)
		meta.Scope = scope
		if strings.TrimSpace(meta.Title) == "" {
			meta.Title = issue.Title
		}
		desc := beads.RenderMeta(meta)
		if body := strings.TrimSpace(beads.StripMeta(issue.Description)); body != "" {
			desc += "\n\n" + body
		}
		updated, err := client.UpdateDescription(nil, issue.ID, desc)
		if err == nil {
			fmt.Printf("Updated task %s\n", updated.ID)
			return nil
		}
		if !errors.Is(err, beads.ErrUpdateDescriptionUnsupported) {
			return fmt.Errorf("updating task %s description: %w", taskID, err)
		}
		deps := append([]string{}, issue.Deps...)
		if !hasDepPrefix(deps, "related:"+issue.ID) {
			deps = append(deps, "related:"+issue.ID)
		}
		replacement, err := client.Create(nil, beads.CreateRequest{
			Title:       issue.Title,
			Type:        "task",
			Priority:    issue.Priority,
			Status:      issue.Status,
			Description: desc,
			Deps:        deps,
		})
		if err != nil {
			return fmt.Errorf("creating replacement task: %w", err)
		}
		_, _ = client.Close(nil, issue.ID, "superseded by "+replacement.ID)
		fmt.Printf("Updated task %s -> %s\n", issue.ID, replacement.ID)
		return nil

	case "list":
		if len(rest) != 1 {
			return fmt.Errorf("usage: mforge task list <rig>")
		}
		rigName := rest[0]
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return fmt.Errorf("loading rig %s: %w", rigName, err)
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		tasks, err := client.List(nil)
		if err != nil {
			return fmt.Errorf("listing tasks: %w", err)
		}
		activeAssignments := map[string]bool{}
		for _, issue := range tasks {
			if strings.ToLower(issue.Type) != "assignment" {
				continue
			}
			if issue.Status == "closed" || issue.Status == "done" {
				continue
			}
			for _, dep := range issue.Deps {
				if strings.HasPrefix(dep, "related:") {
					activeAssignments[strings.TrimPrefix(dep, "related:")] = true
				}
			}
		}
		for _, t := range tasks {
			if strings.ToLower(t.Type) != "task" {
				continue
			}
			meta := beads.ParseMeta(t.Description)
			scope := meta.Scope
			kind := meta.Kind
			assigned := activeAssignments[t.ID]
			scopeLabel := scope
			if scopeLabel == "" {
				scopeLabel = "global"
			}
			fmt.Printf("%s\t%s\t%s\t%s", t.ID, t.Status, kind, strings.ReplaceAll(t.Title, "\t", " "))
			fmt.Printf("\t(scope=%s)\t(assigned=%s)", scopeLabel, yesNo(assigned))
			fmt.Println()
		}
		return nil

	case "split":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task split <rig> --task <id> --cells <a,b,c>")
		}
		rigName := rest[0]
		var taskID, cellsCSV string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			case "--cells":
				if i+1 < len(rest) {
					cellsCSV = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(taskID) == "" || strings.TrimSpace(cellsCSV) == "" {
			return fmt.Errorf("--task and --cells are required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return fmt.Errorf("loading rig %s: %w", rigName, err)
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		parent, err := client.Show(nil, taskID)
		if err != nil {
			return fmt.Errorf("showing parent task %s: %w", taskID, err)
		}
		parentMeta := beads.ParseMeta(parent.Description)
		cells := strings.Split(cellsCSV, ",")
		for _, c := range cells {
			cellName := strings.TrimSpace(c)
			if cellName == "" {
				continue
			}
			cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
			if err != nil {
				return fmt.Errorf("loading cell %s: %w", cellName, err)
			}
			title := fmt.Sprintf("Split: %s (%s)", parent.Title, cellName)
			body := fmt.Sprintf("Parent task: %s\n\n%s", parent.ID, beads.StripMeta(parent.Description))
			scope := cellCfg.ScopePrefix
			meta := beads.Meta{Cell: cellName, Scope: scope, Kind: parentMeta.Kind, Title: title}
			desc := beads.RenderMeta(meta) + "\n\n" + body
			child, err := client.Create(nil, beads.CreateRequest{
				Title:       title,
				Type:        "task",
				Priority:    parent.Priority,
				Status:      "open",
				Description: desc,
				Deps:        []string{"related:" + parent.ID},
			})
			if err != nil {
				return fmt.Errorf("creating split task for %s: %w", cellName, err)
			}
			fmt.Printf("Created child task %s for %s\n", child.ID, cellName)
		}
		return nil

	case "decompose":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task decompose <rig> --task <id> --titles <a,b,c> [--kind <kind>]")
		}
		rigName := rest[0]
		var taskID, titlesCSV, kind string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			case "--titles":
				if i+1 < len(rest) {
					titlesCSV = rest[i+1]
					i++
				}
			case "--kind":
				if i+1 < len(rest) {
					kind = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(taskID) == "" || strings.TrimSpace(titlesCSV) == "" {
			return fmt.Errorf("--task and --titles are required")
		}
		if strings.TrimSpace(kind) == "" {
			kind = "plan"
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return fmt.Errorf("loading rig %s: %w", rigName, err)
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issue, err := client.Show(nil, taskID)
		if err != nil {
			return fmt.Errorf("showing task %s: %w", taskID, err)
		}
		if strings.ToLower(issue.Type) != "task" {
			return fmt.Errorf("bead %s is not a task (type=%s)", issue.ID, issue.Type)
		}
		parentMeta := beads.ParseMeta(issue.Description)
		titles := splitCSV(titlesCSV)
		created := 0
		for _, title := range titles {
			meta := beads.Meta{Scope: parentMeta.Scope, Kind: kind, Title: title}
			desc := beads.RenderMeta(meta)
			if _, err := client.Create(nil, beads.CreateRequest{
				Title:       title,
				Type:        "task",
				Priority:    issue.Priority,
				Status:      "open",
				Description: desc,
				Deps:        []string{"related:" + issue.ID},
			}); err != nil {
				return fmt.Errorf("creating decomposed task %q: %w", title, err)
			}
			created++
		}
		fmt.Printf("Decomposed task %s into %d task(s)\n", taskID, created)
		return nil

	case "complete":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task complete <rig> --task <id> [--reason <text>] [--force]")
		}
		rigName := rest[0]
		var taskID, reason string
		force := false
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			case "--reason":
				if i+1 < len(rest) {
					reason = rest[i+1]
					i++
				}
			case "--force":
				force = true
			}
		}
		if strings.TrimSpace(taskID) == "" {
			return fmt.Errorf("--task is required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return fmt.Errorf("loading rig %s: %w", rigName, err)
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issue, err := client.Show(nil, taskID)
		if err != nil {
			return fmt.Errorf("showing task %s: %w", taskID, err)
		}
		if strings.ToLower(issue.Type) != "task" {
			return fmt.Errorf("bead %s is not a task (type=%s)", issue.ID, issue.Type)
		}
		updated, err := client.CloseWithOptions(nil, issue.ID, beads.CloseOptions{
			Force:  force,
			Reason: reason,
		})
		if err != nil {
			return fmt.Errorf("completing task %s: %w", taskID, err)
		}
		meta := beads.ParseMeta(issue.Description)
		meta.Kind = "task_complete"
		meta.Title = issue.Title
		emitOrchestrationEvent(cfg.RepoPath, meta, fmt.Sprintf("Task completed %s", updated.ID), []string{"related:" + updated.ID})
		fmt.Printf("Completed task %s\n", updated.ID)
		return nil

	case "delete":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task delete <rig> --task <id> [--reason <text>] [--force] [--cascade] [--hard] [--dry-run]")
		}
		rigName := rest[0]
		var taskID, reason string
		force := false
		cascade := false
		hard := false
		dryRun := false
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			case "--reason":
				if i+1 < len(rest) {
					reason = rest[i+1]
					i++
				}
			case "--force":
				force = true
			case "--cascade":
				cascade = true
			case "--hard":
				hard = true
			case "--dry-run":
				dryRun = true
			}
		}
		if strings.TrimSpace(taskID) == "" {
			return fmt.Errorf("--task is required")
		}
		if !force && !dryRun {
			return fmt.Errorf("--force or --dry-run is required (bd delete is destructive)")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return fmt.Errorf("loading rig %s: %w", rigName, err)
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		issue, err := client.Show(nil, taskID)
		if err != nil {
			return fmt.Errorf("showing task %s: %w", taskID, err)
		}
		if strings.ToLower(issue.Type) != "task" {
			return fmt.Errorf("bead %s is not a task (type=%s)", issue.ID, issue.Type)
		}
		out, err := client.Delete(nil, []string{issue.ID}, beads.DeleteOptions{
			Force:   force,
			Hard:    hard,
			Cascade: cascade,
			DryRun:  dryRun,
			Reason:  reason,
		})
		if err != nil {
			return fmt.Errorf("deleting task %s: %w", taskID, err)
		}
		if strings.TrimSpace(out) != "" {
			fmt.Print(out)
			if !strings.HasSuffix(out, "\n") {
				fmt.Println()
			}
		}
		if dryRun {
			fmt.Printf("Delete preview for task %s\n", issue.ID)
			return nil
		}
		fmt.Printf("Deleted task %s\n", issue.ID)
		return nil

	default:
		return fmt.Errorf("unknown task subcommand: %s", op)
	}
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func hasDepPrefix(deps []string, needle string) bool {
	for _, dep := range deps {
		if dep == needle {
			return true
		}
	}
	return false
}
