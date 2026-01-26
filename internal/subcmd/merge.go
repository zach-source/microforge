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

func Merge(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge merge run <rig> [--turn <id>] [--as merge-manager] [--dry-run]")
	}
	op := args[0]
	rest := args[1:]
	if op != "run" {
		return fmt.Errorf("unknown merge subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge merge run <rig> [--turn <id>] [--as merge-manager] [--dry-run]")
	}
	rigName := rest[0]
	turnID := ""
	asRole := ""
	dryRun := false
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--turn":
			if i+1 < len(rest) {
				turnID = rest[i+1]
				i++
			}
		case "--as":
			if i+1 < len(rest) {
				asRole = rest[i+1]
				i++
			}
		case "--dry-run":
			dryRun = true
		}
	}
	if asRole != "merge-manager" {
		return fmt.Errorf("merge requires --as merge-manager")
	}
	if strings.TrimSpace(turnID) == "" {
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
	}
	if strings.TrimSpace(turnID) == "" {
		return fmt.Errorf("no active turn; start a turn or pass --turn")
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
	issueMap := map[string]beads.Issue{}
	for _, issue := range issues {
		issueMap[issue.ID] = issue
	}
	ready, err := client.Ready(nil)
	if err != nil {
		return err
	}
	merged := []string{}
	conflicts := 0
	for _, issue := range ready {
		if strings.ToLower(issue.Type) != "pr" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if meta.TurnID != "" && meta.TurnID != turnID {
			continue
		}
		if meta.Conflict {
			if err := emitConflict(client, meta.Cell, issue.ID, turnID, "merge conflict flagged"); err != nil {
				return err
			}
			conflicts++
			continue
		}
		if !reviewDepsSatisfied(issue, issueMap) {
			continue
		}
		if dryRun {
			merged = append(merged, issue.ID)
			continue
		}
		_, _ = client.Close(nil, issue.ID, "merged")
		merged = append(merged, issue.ID)
	}
	if len(merged) > 0 {
		notePath := filepath.Join(rig.RigDir(home, rigName), fmt.Sprintf("turn-release-notes-%s.md", turnID))
		_ = util.EnsureDir(filepath.Dir(notePath))
		lines := []string{fmt.Sprintf("# Turn %s Release Notes", turnID), "", "Merged PRs:"}
		for _, id := range merged {
			lines = append(lines, "- "+id)
		}
		_ = os.WriteFile(notePath, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	}
	fmt.Printf("Merged %d PR(s)", len(merged))
	if conflicts > 0 {
		fmt.Printf(" (%d conflict(s))", conflicts)
	}
	fmt.Println()
	return nil
}

func reviewDepsSatisfied(issue beads.Issue, issueMap map[string]beads.Issue) bool {
	for _, dep := range issue.Deps {
		if strings.HasPrefix(dep, "review:") {
			revID := strings.TrimPrefix(dep, "review:")
			rev, ok := issueMap[revID]
			if !ok {
				return false
			}
			if rev.Status != "closed" && rev.Status != "done" {
				return false
			}
		}
	}
	return true
}

func emitConflict(client beads.Client, cell, sourceID, turnID, details string) error {
	meta := beads.Meta{Cell: cell, TurnID: turnID, Severity: "high"}
	_, err := client.Create(nil, beads.CreateRequest{
		Title:       "ConflictResolution for " + sourceID,
		Type:        "conflictresolution",
		Priority:    "p1",
		Status:      "open",
		Description: beads.RenderMeta(meta) + "\n\n" + details,
		Deps:        []string{"related:" + sourceID},
	})
	return err
}
