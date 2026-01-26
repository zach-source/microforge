package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Contract(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge contract create <rig> --title <title> --cell <cell> --scope <path> [--acceptance <text>] [--compat <text>] [--links <text>]")
	}
	op := args[0]
	rest := args[1:]
	if op != "create" {
		return fmt.Errorf("unknown contract subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge contract create <rig> --title <title> --cell <cell> --scope <path> [--acceptance <text>] [--compat <text>] [--links <text>]")
	}
	rigName := rest[0]
	var title, cell, scope, acceptance, compat, links, turnID string
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--title":
			if i+1 < len(rest) {
				title = rest[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(rest) {
				cell = rest[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(rest) {
				scope = rest[i+1]
				i++
			}
		case "--acceptance":
			if i+1 < len(rest) {
				acceptance = rest[i+1]
				i++
			}
		case "--compat":
			if i+1 < len(rest) {
				compat = rest[i+1]
				i++
			}
		case "--links":
			if i+1 < len(rest) {
				links = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(title) == "" || strings.TrimSpace(cell) == "" || strings.TrimSpace(scope) == "" {
		return fmt.Errorf("--title, --cell, and --scope are required")
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
	body := "## Acceptance Criteria\n" + defaultIfEmpty(acceptance, "- [ ] Define contract tests\n- [ ] Document rollout plan")
	body += "\n\n## Compatibility Notes\n" + defaultIfEmpty(compat, "Additive first; deprecation window if needed.")
	if strings.TrimSpace(links) != "" {
		body += "\n\n## Links\n" + links
	}
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "contract",
		Priority:    "p1",
		Status:      "open",
		Description: beads.RenderMeta(meta) + "\n\n" + body,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Created contract %s\n", issue.ID)
	return nil
}
