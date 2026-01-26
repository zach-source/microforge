package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Coordinator(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge coordinator sync <rig> [--turn <id>]")
	}
	op := args[0]
	rest := args[1:]
	if op != "sync" {
		return fmt.Errorf("unknown coordinator subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge coordinator sync <rig> [--turn <id>]")
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
	count := 0
	for _, issue := range issues {
		meta := beads.ParseMeta(issue.Description)
		if turnID != "" && meta.TurnID != "" && meta.TurnID != turnID {
			continue
		}
		if strings.ToLower(issue.Type) == "pr" && issue.Status == "ready" {
			_, _ = client.UpdateStatus(nil, issue.ID, "queued")
			count++
		}
	}
	fmt.Printf("Coordinator sync queued %d PR(s) for merge\n", count)
	return nil
}
