package subcmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Wait(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge wait <rig> [--turn <id>] [--interval <seconds>]")
	}
	rigName := args[0]
	turnID := ""
	interval := 2 * time.Second
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--turn":
			if i+1 < len(args) {
				turnID = args[i+1]
				i++
			}
		case "--interval":
			if i+1 < len(args) {
				if secs, err := time.ParseDuration(args[i+1] + "s"); err == nil {
					interval = secs
				}
				i++
			}
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
	for {
		issues, err := client.List(nil)
		if err != nil {
			return err
		}
		open := 0
		for _, issue := range issues {
			if strings.ToLower(issue.Type) != "assignment" {
				continue
			}
			meta := beads.ParseMeta(issue.Description)
			if turnID != "" && meta.TurnID != "" && meta.TurnID != turnID {
				continue
			}
			if issue.Status == "closed" || issue.Status == "done" {
				continue
			}
			open++
		}
		if open == 0 {
			fmt.Println("All assignments complete")
			return nil
		}
		fmt.Printf("Waiting on %d assignment(s)...\n", open)
		time.Sleep(interval)
	}
}
