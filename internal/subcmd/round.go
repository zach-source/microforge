package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
	"github.com/example/microforge/internal/util"
)

func Round(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge round <start|review|merge> [--wait]")
	}
	op := args[0]
	rest := args[1:]
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge round %s [--wait]", op)
	}
	rigName := rest[0]
	wait := false
	feature := ""
	base := ""
	all := false
	changesOnly := false
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--wait":
			wait = true
		case "--feature":
			if i+1 < len(rest) {
				feature = rest[i+1]
				i++
			}
		case "--base":
			if i+1 < len(rest) {
				base = rest[i+1]
				i++
			}
		case "--all":
			all = true
		case "--changes-only":
			changesOnly = true
		}
	}
	switch op {
	case "start":
		return roundStart(home, rigName, wait)
	case "review":
		return roundReview(home, rigName, wait, base, all, changesOnly)
	case "merge":
		if strings.TrimSpace(feature) == "" {
			return fmt.Errorf("usage: mforge round merge <rig> --feature <branch> [--base <branch>]")
		}
		return roundMerge(home, rigName, feature, base)
	default:
		return fmt.Errorf("unknown round subcommand: %s", op)
	}
}

func roundStart(home, rigName string, wait bool) error {
	turnID, err := ensureTurn(home, rigName)
	if err != nil {
		return err
	}
	warnContextMismatch(home, rigName, "round start")
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
	assignedTasks := existingAssignments(issues)
	wakeSet := map[string]struct{}{}
	assigned := 0
	for _, issue := range issues {
		typeLower := strings.ToLower(issue.Type)
		if typeLower != "task" && typeLower != "request" && typeLower != "observation" {
			continue
		}
		if issue.Status == "closed" || issue.Status == "done" {
			continue
		}
		if assignedTasks[issue.ID] {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		cell := matchCellByScope(cells, meta.Scope)
		if cell == nil {
			continue
		}
		if err := beadLimit(home, rigName, cell.Name, turnID); err != nil {
			return err
		}
		role := roleForTask(*cell, meta)
		if err := ensureCellBootstrapped(home, rigName, cell.Name, role, true); err != nil {
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
		wakeSet[cell.Name+"|"+role] = struct{}{}
		assigned++
	}
	for key := range wakeSet {
		parts := strings.Split(key, "|")
		cell, role := parts[0], parts[1]
		if err := Agent(home, []string{"wake", rigName, cell, role}); err != nil {
			fmt.Printf("Wake skipped for %s/%s: %v\n", cell, role, err)
		}
	}
	if wait {
		if err := Wait(home, []string{rigName}); err != nil {
			return err
		}
	}
	if _, err := reconcile(home, rigName, false); err != nil {
		return err
	}
	emitOrchestrationEvent(cfg.RepoPath, beads.Meta{Kind: "round_start"}, fmt.Sprintf("Round start %s", rigName), nil)
	fmt.Printf("Round start: assigned %d task(s)\n", assigned)
	return nil
}

func roundReview(home, rigName string, wait bool, base string, all bool, changesOnly bool) error {
	turnID, err := ensureTurn(home, rigName)
	if err != nil {
		return err
	}
	warnContextMismatch(home, rigName, "round review")
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	if base == "" {
		base = detectBaseBranch(cfg.RepoPath)
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
	existing := reviewByCell(issues, turnID)
	wakeSet := map[string]struct{}{}
	created := 0
	skipped := 0
	considerChanges := !all || changesOnly
	for _, cell := range cells {
		if existing[cell.Name] {
			continue
		}
		if considerChanges {
			ok, reason := cellHasChanges(cfg.RepoPath, cell.WorktreePath, base)
			if !ok {
				fmt.Printf("Skipping review for %s: %s\n", cell.Name, reason)
				skipped++
				continue
			}
		}
		role := roleForReview(cell)
		if err := ensureCellBootstrapped(home, rigName, cell.Name, role, true); err != nil {
			return err
		}
		if err := beadLimit(home, rigName, cell.Name, turnID); err != nil {
			return err
		}
		title := fmt.Sprintf("Round review %s", cell.Name)
		meta := beads.Meta{Cell: cell.Name, Scope: cell.ScopePrefix, TurnID: turnID, Role: role, Kind: "review"}
		reviewIssue, err := client.Create(nil, beads.CreateRequest{
			Title:       title,
			Type:        "review",
			Priority:    "p2",
			Status:      "open",
			Description: beads.RenderMeta(meta) + "\n\nReview decisions from this round.",
		})
		if err != nil {
			return err
		}
		inboxRel := fmt.Sprintf("mail/inbox/%s.md", reviewIssue.ID)
		outboxRel := fmt.Sprintf("mail/outbox/%s.md", reviewIssue.ID)
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
		mail, _ := writeAssignmentInbox(cell.WorktreePath, inboxRel, outboxRel, "DONE", reviewIssue)
		_ = createMailBead(client, assnMeta, "Mail "+assn.ID, mail, []string{"related:" + assn.ID})
		if err := Agent(home, []string{"spawn", rigName, cell.Name, role}); err != nil {
			fmt.Printf("Spawn skipped for %s/%s: %v\n", cell.Name, role, err)
		}
		wakeSet[cell.Name+"|"+role] = struct{}{}
		created++
	}
	for key := range wakeSet {
		parts := strings.Split(key, "|")
		cell, role := parts[0], parts[1]
		if err := Agent(home, []string{"wake", rigName, cell, role}); err != nil {
			fmt.Printf("Wake skipped for %s/%s: %v\n", cell, role, err)
		}
	}
	if wait {
		if err := Wait(home, []string{rigName}); err != nil {
			return err
		}
	}
	if _, err := reconcile(home, rigName, false); err != nil {
		return err
	}
	emitOrchestrationEvent(cfg.RepoPath, beads.Meta{Kind: "round_review"}, fmt.Sprintf("Round review %s", rigName), nil)
	if skipped > 0 {
		fmt.Printf("Round review: created %d review task(s); skipped %d (no changes)\n", created, skipped)
	} else {
		fmt.Printf("Round review: created %d review task(s)\n", created)
	}
	return nil
}

func detectBaseBranch(repo string) string {
	candidates := []string{"main", "master"}
	for _, c := range candidates {
		if _, err := util.Run(nil, "git", "-C", repo, "rev-parse", "--verify", c); err == nil {
			return c
		}
	}
	return "HEAD"
}

func cellHasChanges(repo, worktree, base string) (bool, string) {
	if _, err := os.Stat(filepath.Join(worktree, ".git")); err != nil {
		return true, "no git worktree detected"
	}
	res, err := util.Run(nil, "git", "-C", worktree, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return true, "unable to determine branch"
	}
	branch := strings.TrimSpace(res.Stdout)
	if branch == "" || branch == "HEAD" {
		return true, "detached HEAD"
	}
	if base == "" {
		base = "HEAD"
	}
	if _, err := util.Run(nil, "git", "-C", repo, "diff", "--quiet", base+".."+branch); err == nil {
		return false, "no changes"
	}
	return true, "changes detected"
}

func roundMerge(home, rigName, feature, base string) error {
	warnContextMismatch(home, rigName, "round merge")
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	if base == "" {
		base = "HEAD"
	}
	if res, err := util.Run(nil, "git", "-C", cfg.RepoPath, "status", "--porcelain"); err == nil {
		if strings.TrimSpace(res.Stdout) != "" {
			return fmt.Errorf("repo has uncommitted changes, aborting merge")
		}
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	type branchRef struct {
		Cell   string
		Branch string
	}
	branches := []branchRef{}
	seen := map[string]bool{}
	for _, cell := range cells {
		if _, err := os.Stat(filepath.Join(cell.WorktreePath, ".git")); err != nil {
			continue
		}
		res, err := util.Run(nil, "git", "-C", cell.WorktreePath, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			continue
		}
		branch := strings.TrimSpace(res.Stdout)
		if branch == "" || branch == "HEAD" || branch == feature {
			continue
		}
		if seen[branch] {
			continue
		}
		seen[branch] = true
		branches = append(branches, branchRef{Cell: cell.Name, Branch: branch})
	}
	if len(branches) == 0 {
		fmt.Println("No cell branches to merge")
		return nil
	}
	if _, err := util.Run(nil, "git", "-C", cfg.RepoPath, "checkout", "-B", feature, base); err != nil {
		return err
	}
	for _, ref := range branches {
		msg := fmt.Sprintf("Merge %s (%s)", ref.Branch, ref.Cell)
		if _, err := util.Run(nil, "git", "-C", cfg.RepoPath, "merge", "--no-ff", ref.Branch, "-m", msg); err != nil {
			return err
		}
	}
	fmt.Printf("Merged %d branch(es) into %s\n", len(branches), feature)
	return nil
}

func ensureTurn(home, rigName string) (string, error) {
	state, err := turn.Load(rig.TurnStatePath(home, rigName))
	if err == nil {
		return strings.TrimSpace(state.ID), nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if err := Turn(home, []string{"start", rigName}); err != nil {
		return "", err
	}
	state, err = turn.Load(rig.TurnStatePath(home, rigName))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(state.ID), nil
}

func existingAssignments(issues []beads.Issue) map[string]bool {
	out := map[string]bool{}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "assignment" {
			continue
		}
		for _, dep := range issue.Deps {
			if strings.HasPrefix(dep, "related:") {
				taskID := strings.TrimPrefix(dep, "related:")
				if strings.TrimSpace(taskID) != "" {
					out[taskID] = true
				}
			}
		}
	}
	return out
}

func reviewByCell(issues []beads.Issue, turnID string) map[string]bool {
	out := map[string]bool{}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "review" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.TrimSpace(meta.Cell) == "" {
			continue
		}
		if turnID != "" && strings.TrimSpace(meta.TurnID) != turnID {
			continue
		}
		if issue.Status == "closed" || issue.Status == "done" {
			continue
		}
		out[meta.Cell] = true
	}
	return out
}

func roleForTask(cell rig.CellConfig, meta beads.Meta) string {
	if strings.TrimSpace(meta.Role) != "" {
		return meta.Role
	}
	if isDocKind(meta.Kind) && roleExists(cell.WorktreePath, "architect") {
		return "architect"
	}
	if roleExists(cell.WorktreePath, "cell") {
		return "cell"
	}
	return "builder"
}

func roleForReview(cell rig.CellConfig) string {
	if roleExists(cell.WorktreePath, "cell") {
		return "cell"
	}
	if roleExists(cell.WorktreePath, "reviewer") {
		return "reviewer"
	}
	if roleExists(cell.WorktreePath, "architect") {
		return "architect"
	}
	return "builder"
}

func roleExists(worktree, role string) bool {
	if strings.TrimSpace(worktree) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(worktree, ".mf", "active-agent-"+role+".json"))
	return err == nil
}

func isDocKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "doc", "docs", "plan", "design":
		return true
	default:
		return false
	}
}
