package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
)

func Convoy(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge convoy <start> --epic <id> [--role <role>] [--title <text>]")
	}
	op := args[0]
	rest := args[1:]
	if op != "start" {
		return fmt.Errorf("unknown convoy subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge convoy start --epic <id> [--role <role>] [--title <text>]")
	}
	rigName := rest[0]
	epicID := ""
	role := "builder"
	title := ""
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--epic":
			if i+1 < len(rest) {
				epicID = rest[i+1]
				i++
			}
		case "--role":
			if i+1 < len(rest) {
				role = rest[i+1]
				i++
			}
		case "--title":
			if i+1 < len(rest) {
				title = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(epicID) == "" {
		return fmt.Errorf("--epic is required")
	}
	if strings.TrimSpace(title) == "" {
		title = "Convoy for epic " + epicID
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	meta := beads.Meta{Kind: "convoy", Title: title}
	convoy, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "convoy",
		Priority:    "p2",
		Status:      "open",
		Description: beads.RenderMeta(meta),
		Deps:        []string{"related:" + epicID},
	})
	if err != nil {
		return err
	}
	if err := Epic(home, []string{"assign", rigName, "--epic", epicID, "--role", role}); err != nil {
		return err
	}
	issues, err := client.List(nil)
	if err != nil {
		return err
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	wakeSet := map[string]struct{}{}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "task" {
			continue
		}
		if !hasDep(issue, epicID) {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		cell := matchCellByScope(cells, meta.Scope)
		if cell == nil {
			continue
		}
		wakeSet[cell.Name+"|"+role] = struct{}{}
	}
	for key := range wakeSet {
		parts := strings.Split(key, "|")
		cell, role := parts[0], parts[1]
		_ = Agent(home, []string{"wake", rigName, cell, role})
	}
	emitOrchestrationEvent(cfg.RepoPath, beads.Meta{Kind: "convoy_start", ConvoyID: convoy.ID}, "Convoy start "+convoy.ID, []string{"related:" + epicID})
	fmt.Printf("Convoy started %s\n", convoy.ID)
	return nil
}
