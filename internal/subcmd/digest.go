package subcmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Digest(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge digest render <rig> [--turn <id>]")
	}
	op := args[0]
	rest := args[1:]
	if op != "render" {
		return fmt.Errorf("unknown digest subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge digest render <rig> [--turn <id>]")
	}
	rigName := rest[0]
	turnID := ""
	for i := 1; i < len(rest); i++ {
		if rest[i] == "--turn" && i+1 < len(rest) {
			turnID = rest[i+1]
			i++
		}
	}
	if strings.TrimSpace(turnID) == "" {
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
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

	closedByType := map[string][]string{}
	openDecisions := []string{}
	observations := []string{}
	mergedPRs := []string{}
	for _, issue := range issues {
		meta := beads.ParseMeta(issue.Description)
		if turnID != "" && meta.TurnID != "" && meta.TurnID != turnID {
			continue
		}
		typeLower := strings.ToLower(issue.Type)
		switch typeLower {
		case "pr":
			if issue.Status == "closed" || issue.Status == "done" {
				mergedPRs = append(mergedPRs, issue.ID+" "+issue.Title)
			}
		case "decision":
			if issue.Status != "closed" && issue.Status != "done" {
				openDecisions = append(openDecisions, issue.ID+" "+issue.Title)
			}
		case "observation":
			if issue.Status != "closed" && issue.Status != "done" {
				observations = append(observations, issue.ID+" "+issue.Title)
			}
		}
		if issue.Status == "closed" || issue.Status == "done" {
			closedByType[typeLower] = append(closedByType[typeLower], issue.ID+" "+issue.Title)
		}
	}

	fmt.Printf("Turn Digest: %s\n", turnID)
	if len(mergedPRs) > 0 {
		fmt.Println("\nMerged PRs")
		for _, line := range mergedPRs {
			fmt.Println("- " + line)
		}
	}
	if len(closedByType) > 0 {
		fmt.Println("\nClosed Beads")
		keys := make([]string, 0, len(closedByType))
		for k := range closedByType {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%s\n", k)
			for _, line := range closedByType[k] {
				fmt.Println("- " + line)
			}
		}
	}
	if len(observations) > 0 {
		fmt.Println("\nOpen Observations")
		for _, line := range observations {
			fmt.Println("- " + line)
		}
	}
	if len(openDecisions) > 0 {
		fmt.Println("\nDecisions Needed")
		for _, line := range openDecisions {
			fmt.Println("- " + line)
		}
	}
	return nil
}
