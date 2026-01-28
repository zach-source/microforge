package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
)

func QuickAssign(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge quick-assign <rig> <bead-id> <cell> [--role <role>] [--promise <token>]")
	}
	rigName := args[0]
	if len(args) < 3 {
		return fmt.Errorf("usage: mforge quick-assign <rig> <bead-id> <cell> [--role <role>] [--promise <token>]")
	}
	beadID := args[1]
	cellName := args[2]
	role := "builder"
	promise := "DONE"
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--promise":
			if i+1 < len(args) {
				promise = args[i+1]
				i++
			}
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issue, err := client.Show(nil, beadID)
	if err != nil {
		return err
	}
	if strings.ToLower(issue.Type) == "assignment" {
		return fmt.Errorf("bead %s is already an assignment", beadID)
	}
	argsAssign := []string{rigName, "--task", beadID, "--cell", cellName, "--role", role, "--promise", promise, "--quick"}
	return Assign(home, argsAssign)
}
