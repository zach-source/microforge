package subcmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Architect(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge architect <docs|contract|design> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "docs":
		return architectRequest(home, rest, "Docs update required")
	case "contract":
		return architectRequest(home, rest, "Contract check required")
	case "design":
		return architectRequest(home, rest, "Design review required")
	default:
		return fmt.Errorf("unknown architect subcommand: %s", op)
	}
}

func architectRequest(home string, args []string, title string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge architect <docs|contract|design> <rig> --cell <cell> --details <text> [--scope <path>]")
	}
	rigName := args[0]
	var cellName, details, scope string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--cell":
			if i+1 < len(args) {
				cellName = args[i+1]
				i++
			}
		case "--details":
			if i+1 < len(args) {
				details = args[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(args) {
				scope = args[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(cellName) == "" || strings.TrimSpace(details) == "" {
		return fmt.Errorf("--cell and --details are required")
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	turnID := ""
	if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
		turnID = strings.TrimSpace(state.ID)
	}
	if err := beadLimit(home, rigName, cellName, turnID); err != nil {
		return err
	}
	payload := map[string]any{
		"title":   title,
		"details": details,
		"scope":   scope,
	}
	payloadJSON, _ := json.Marshal(payload)
	meta := beads.Meta{Cell: cellName, SourceRole: "architect", Scope: scope, TurnID: turnID}
	_, err = client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "request",
		Priority:    "p2",
		Status:      "open",
		Description: beads.RenderMeta(meta) + "\n\n" + string(payloadJSON),
	})
	if err != nil {
		return err
	}
	fmt.Println("Created architect request")
	return nil
}
