package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func SSH(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge ssh <rig> --cmd <command...> [--tty]")
	}
	rigName := args[0]
	var cmdParts []string
	tty := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--cmd":
			for j := i + 1; j < len(args); j++ {
				cmdParts = append(cmdParts, args[j])
			}
			i = len(args)
		case "--tty":
			tty = true
		}
	}
	if len(cmdParts) == 0 {
		return fmt.Errorf("--cmd is required")
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.RemoteHost) == "" {
		return fmt.Errorf("remote_host is not configured")
	}
	cmd, args := buildSSHCommand(cfg, cmdParts, tty)
	_, err = util.Run(nil, cmd, args...)
	return err
}

func buildSSHCommand(cfg rig.RigConfig, remoteCmd []string, tty bool) (string, []string) {
	dest := cfg.RemoteHost
	if strings.TrimSpace(cfg.RemoteUser) != "" {
		dest = cfg.RemoteUser + "@" + dest
	}
	args := []string{}
	if tty {
		args = append(args, "-t")
	}
	if cfg.RemotePort != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", cfg.RemotePort))
	}
	args = append(args, dest)
	args = append(args, remoteCmd...)
	return "ssh", args
}
