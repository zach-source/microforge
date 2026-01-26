package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/rig"
)

func TurnRun(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge turn run <rig> [--role <role>] [--wait]")
	}
	rigName := args[0]
	role := "builder"
	wait := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--wait":
			wait = true
		}
	}
	if err := Turn(home, []string{"start", rigName}); err != nil {
		return err
	}
	if err := ManagerAssign(home, rigName, []string{"--role", role}); err != nil {
		return err
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	for _, cell := range cells {
		if strings.EqualFold(cell.Name, "monitor") {
			continue
		}
		_ = Agent(home, []string{"wake", rigName, cell.Name, role})
	}
	if wait {
		if err := Wait(home, []string{rigName}); err != nil {
			return err
		}
	}
	fmt.Println("Turn run complete. Use `mforge merge run` and `mforge turn end` when ready.")
	return nil
}
