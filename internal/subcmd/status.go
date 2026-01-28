package subcmd

import "fmt"

func Status(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge status <rig> [--cell <cell>] [--role <role>] [--json]")
	}
	return Agent(home, append([]string{"status"}, args...))
}
