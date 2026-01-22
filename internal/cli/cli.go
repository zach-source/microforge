package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/example/microforge/internal/subcmd"
	"github.com/example/microforge/internal/store"
)

func usage() string {
	return strings.TrimSpace(`
mf — Microforge (SQLite + tmux + Claude Code hooks)

Usage:
  mf init <rig> --repo <path>

  mf cell add <rig> <cell> --scope <path-prefix>
  mf cell bootstrap <rig> <cell>

  mf agent spawn <rig> <cell> <role>
  mf agent stop  <rig> <cell> <role>
  mf agent attach <rig> <cell> <role>
  mf agent wake <rig> <cell> <role>

  mf task create <rig> --title <t> [--body <md>] [--scope <path-prefix>] [--kind improve|fix|review|monitor|doc]
  mf task list <rig>

  mf assign <rig> --task <id> --cell <cell> --role builder|monitor|reviewer|architect
  mf manager tick <rig> [--watch]

  # Invoked by Claude Code hooks:
  mf hook stop [--role <role>]
  mf hook guardrails

Environment:
  MF_HOME   override default home (~/.microforge)
`)
}

func Run(args []string) error {
	if len(args) == 0 {
		fmt.Println(usage())
		return nil
	}
	home := store.DefaultHome()
	if v := strings.TrimSpace(os.Getenv("MF_HOME")); v != "" {
		home = v
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "help", "-h", "--help":
		fmt.Println(usage())
		return nil
	case "init":
		return subcmd.Init(home, rest)
	case "cell":
		return subcmd.Cell(home, rest)
	case "agent":
		return subcmd.Agent(home, rest)
	case "task":
		return subcmd.Task(home, rest)
	case "assign":
		return subcmd.Assign(home, rest)
	case "manager":
		return subcmd.Manager(home, rest)
	case "hook":
		return subcmd.Hook(home, rest)
	default:
		return errors.New("unknown command: " + cmd + "\n\n" + usage())
	}
}
