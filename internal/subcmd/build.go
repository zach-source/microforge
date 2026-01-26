package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Build(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge build record <rig> --service <name> --image <tag> [--status <status>] [--turn <id>]")
	}
	op := args[0]
	rest := args[1:]
	if op != "record" {
		return fmt.Errorf("unknown build subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge build record <rig> --service <name> --image <tag> [--status <status>] [--turn <id>]")
	}
	rigName := rest[0]
	var service, image, status, turnID string
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--service":
			if i+1 < len(rest) {
				service = rest[i+1]
				i++
			}
		case "--image":
			if i+1 < len(rest) {
				image = rest[i+1]
				i++
			}
		case "--status":
			if i+1 < len(rest) {
				status = rest[i+1]
				i++
			}
		case "--turn":
			if i+1 < len(rest) {
				turnID = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(service) == "" || strings.TrimSpace(image) == "" {
		return fmt.Errorf("--service and --image are required")
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
	meta := beads.Meta{TurnID: turnID}
	body := fmt.Sprintf("service=%s\nimage=%s\nstatus=%s", service, image, defaultIfEmpty(status, "built"))
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       "Build " + service,
		Type:        "build",
		Priority:    "p2",
		Status:      defaultIfEmpty(status, "done"),
		Description: beads.RenderMeta(meta) + "\n\n" + body,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Recorded build %s\n", issue.ID)
	return nil
}
