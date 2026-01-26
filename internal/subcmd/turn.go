package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
	"github.com/example/microforge/internal/util"
)

func Turn(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge turn <start|status|end|slate> <rig>")
	}
	op := args[0]
	rest := args[1:]
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge turn %s <rig>", op)
	}
	rigName := rest[0]
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	statePath := rig.TurnStatePath(home, rigName)
	switch op {
	case "start":
		title := fmt.Sprintf("Turn %s", time.Now().UTC().Format(time.RFC3339))
		issue, err := client.Create(nil, beads.CreateRequest{
			Title:       title,
			Type:        "turn",
			Priority:    "p1",
			Status:      "open",
			Description: "turn-start",
		})
		if err != nil {
			return err
		}
		state := turn.State{ID: issue.ID, StartedAt: time.Now().UTC().Format(time.RFC3339), Status: "open"}
		if err := util.EnsureDir(filepath.Dir(statePath)); err != nil {
			return err
		}
		if err := turn.Save(statePath, state); err != nil {
			return err
		}
		fmt.Printf("Started turn %s (%s)\n", issue.ID, title)
		return nil
	case "status":
		state, err := turn.Load(statePath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No active turn")
				return nil
			}
			return err
		}
		fmt.Printf("Turn %s (%s) status=%s\n", state.ID, state.StartedAt, strings.TrimSpace(state.Status))
		return nil
	case "end":
		state, err := turn.Load(statePath)
		if err != nil {
			return err
		}
		if strings.TrimSpace(state.ID) != "" {
			_, _ = client.Close(nil, state.ID, "turn complete")
		}
		_ = os.Remove(statePath)
		fmt.Printf("Ended turn %s\n", state.ID)
		return nil
	case "slate":
		state, err := turn.Load(statePath)
		if err != nil {
			return err
		}
		issues, err := client.List(nil)
		if err != nil {
			return err
		}
		fmt.Printf("Turn slate %s\n", state.ID)
		for _, issue := range issues {
			meta := beads.ParseMeta(issue.Description)
			if meta.TurnID != state.ID {
				continue
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", issue.ID, issue.Status, issue.Type, strings.ReplaceAll(issue.Title, "\t", " "))
		}
		return nil
	default:
		return fmt.Errorf("unknown turn subcommand: %s", op)
	}
}
