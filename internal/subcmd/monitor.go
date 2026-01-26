package subcmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
	"github.com/example/microforge/internal/util"
)

func Monitor(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge monitor <run-tests|run> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "run-tests":
		return monitorRunTests(home, rest)
	case "run":
		if len(rest) >= 2 && rest[1] == "--all" {
			return monitorRunGlobal(home, rest[:1])
		}
		if len(rest) >= 2 && rest[1] == "all" {
			return monitorRunGlobal(home, rest[:1])
		}
		if len(rest) >= 2 && rest[1] == "--global" {
			return monitorRunGlobal(home, rest[:1])
		}
		if len(rest) >= 1 && len(rest) < 2 {
			return monitorRunGlobal(home, rest)
		}
		return monitorRun(home, rest)
	default:
		return fmt.Errorf("unknown monitor subcommand: %s", op)
	}
}

func monitorRunTests(home string, rest []string) error {
	if len(rest) < 2 {
		return fmt.Errorf("usage: mforge monitor run-tests <rig> <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>]")
	}
	rigName, cellName := rest[0], rest[1]
	var cmdParts []string
	severity := "high"
	priority := "p1"
	scope := ""
	for i := 2; i < len(rest); i++ {
		switch rest[i] {
		case "--cmd":
			for j := i + 1; j < len(rest); j++ {
				cmdParts = append(cmdParts, rest[j])
			}
			i = len(rest)
		case "--severity":
			if i+1 < len(rest) {
				severity = rest[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(rest) {
				priority = rest[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(rest) {
				scope = rest[i+1]
				i++
			}
		}
	}
	if len(cmdParts) == 0 {
		return fmt.Errorf("--cmd is required")
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}

	cmd := cmdParts[0]
	cmdArgs := cmdParts[1:]
	res, err := util.RunInDir(nil, cellCfg.WorktreePath, cmd, cmdArgs...)
	if err == nil {
		fmt.Println("OK")
		return nil
	}
	if strings.TrimSpace(scope) == "" {
		scope = cellCfg.ScopePrefix
	}
	turnID := ""
	if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
		turnID = strings.TrimSpace(state.ID)
	}
	if err := beadLimit(home, rigName, cellName, turnID); err != nil {
		return err
	}
	title := "Monitor failure: " + strings.Join(cmdParts, " ")
	payload := map[string]any{
		"title":   title,
		"body":    "Command failed: " + strings.Join(cmdParts, " "),
		"kind":    "monitor",
		"scope":   scope,
		"command": strings.Join(cmdParts, " "),
		"stdout":  res.Stdout,
		"stderr":  res.Stderr,
	}
	payloadJSON, _ := json.Marshal(payload)
	meta := beads.Meta{Cell: cellName, SourceRole: "monitor", Scope: scope, Kind: "observation", Severity: severity, TurnID: turnID}
	if _, reqErr := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "observation",
		Priority:    priority,
		Status:      "open",
		Description: beads.RenderMeta(meta) + "\n\n" + string(payloadJSON),
	}); reqErr != nil {
		return reqErr
	}
	fmt.Println("Created observation from monitor failure")
	return nil
}

func monitorRun(home string, rest []string) error {
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge monitor run <rig> <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>] [--observation <text>]")
	}
	rigName, cellName := rest[0], ""
	if len(rest) >= 2 {
		cellName = rest[1]
	}
	var cmdParts []string
	severity := "med"
	priority := "p2"
	scope := ""
	observation := ""
	for i := 2; i < len(rest); i++ {
		switch rest[i] {
		case "--cmd":
			for j := i + 1; j < len(rest); j++ {
				cmdParts = append(cmdParts, rest[j])
			}
			i = len(rest)
		case "--severity":
			if i+1 < len(rest) {
				severity = rest[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(rest) {
				priority = rest[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(rest) {
				scope = rest[i+1]
				i++
			}
		case "--observation":
			if i+1 < len(rest) {
				observation = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(cellName) == "" {
		return fmt.Errorf("cell is required")
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	if strings.TrimSpace(scope) == "" {
		scope = cellCfg.ScopePrefix
	}
	turnID := ""
	if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
		turnID = strings.TrimSpace(state.ID)
	}
	if err := beadLimit(home, rigName, cellName, turnID); err != nil {
		return err
	}
	if len(cmdParts) > 0 {
		cmd := cmdParts[0]
		cmdArgs := cmdParts[1:]
		res, err := util.RunInDir(nil, cellCfg.WorktreePath, cmd, cmdArgs...)
		if err == nil {
			if strings.TrimSpace(observation) == "" {
				fmt.Println("OK")
				return nil
			}
		}
		if strings.TrimSpace(observation) == "" {
			observation = "Monitor signal: " + strings.Join(cmdParts, " ")
		}
		payload := map[string]any{
			"title":   observation,
			"scope":   scope,
			"command": strings.Join(cmdParts, " "),
			"stdout":  res.Stdout,
			"stderr":  res.Stderr,
		}
		payloadJSON, _ := json.Marshal(payload)
		meta := beads.Meta{Cell: cellName, SourceRole: "monitor", Scope: scope, Kind: "observation", Severity: severity, TurnID: turnID}
		_, err = client.Create(nil, beads.CreateRequest{
			Title:       observation,
			Type:        "observation",
			Priority:    priority,
			Status:      "open",
			Description: beads.RenderMeta(meta) + "\n\n" + string(payloadJSON),
		})
		if err != nil {
			return err
		}
		fmt.Println("Created observation")
		return nil
	}
	if strings.TrimSpace(observation) == "" {
		return fmt.Errorf("--cmd or --observation is required")
	}
	meta := beads.Meta{Cell: cellName, SourceRole: "monitor", Scope: scope, Kind: "observation", Severity: severity, TurnID: turnID}
	_, err = client.Create(nil, beads.CreateRequest{
		Title:       observation,
		Type:        "observation",
		Priority:    priority,
		Status:      "open",
		Description: beads.RenderMeta(meta) + "\n\n" + observation,
	})
	if err != nil {
		return err
	}
	fmt.Println("Created observation")
	return nil
}
