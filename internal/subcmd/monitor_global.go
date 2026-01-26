package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/rig"
)

func monitorRunGlobal(home string, rest []string) error {
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge monitor run <rig> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>] [--observation <text>]")
	}
	rigName := rest[0]
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	for _, cell := range cells {
		if strings.EqualFold(cell.Name, "monitor") {
			continue
		}
		args := append([]string{rigName, cell.Name}, rest[1:]...)
		if err := monitorRun(home, args); err != nil {
			return err
		}
	}
	fmt.Println("Global monitor run complete")
	return nil
}
