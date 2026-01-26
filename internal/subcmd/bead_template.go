package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func BeadTemplate(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge bead template <rig> --type <type> --title <title> --cell <cell> [--scope <path>] [--priority <p>] [--turn <id>]")
	}
	rigName := args[0]
	var beadType, title, cell, scope, priority, turnID string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				beadType = args[i+1]
				i++
			}
		case "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(args) {
				cell = args[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(args) {
				scope = args[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(args) {
				priority = args[i+1]
				i++
			}
		case "--turn":
			if i+1 < len(args) {
				turnID = args[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(beadType) == "" || strings.TrimSpace(title) == "" || strings.TrimSpace(cell) == "" {
		return fmt.Errorf("--type, --title, and --cell are required")
	}
	if strings.TrimSpace(turnID) == "" {
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
	}
	if err := beadLimit(home, rigName, cell, turnID); err != nil {
		return err
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	meta := beads.Meta{Cell: cell, Scope: scope, TurnID: turnID}
	body := renderTemplate(beadType, "", "", "", "")
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        beadType,
		Priority:    defaultIfEmpty(priority, "p2"),
		Status:      "open",
		Description: beads.RenderMeta(meta) + "\n\n" + body,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Created template bead %s\n", issue.ID)
	return nil
}
