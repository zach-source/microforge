package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/subcmd"
)

func usage() string {
	return strings.TrimSpace(`
mforge — Microforge (SQLite + tmux + Claude Code hooks)

Usage:
  mforge init <rig> --repo <path>

  mforge cell add <rig> <cell> --scope <path-prefix>
  mforge cell bootstrap <rig> <cell> [--architect]

  mforge agent spawn <rig> <cell> <role>
  mforge agent stop  <rig> <cell> <role>
  mforge agent attach <rig> <cell> <role>
  mforge agent wake <rig> <cell> <role>
  mforge agent status <rig> [--cell <cell>] [--role <role>] [--remote]

  mforge task create <rig> --title <t> [--body <md>] [--scope <path-prefix>] [--kind improve|fix|review|monitor|doc]
  mforge task list <rig>

  mforge assign <rig> --task <id> --cell <cell> --role builder|monitor|reviewer|architect [--promise <token>]
  mforge request create <rig> --cell <cell> --role <role> --severity <sev> --priority <p> --scope <path> --payload <json>
  mforge request list <rig> [--cell <cell>] [--status <status>] [--priority <p>]
  mforge request triage <rig> --request <id> --action create-task|merge|block
  mforge monitor run-tests <rig> <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>]
  mforge epic create <rig> --title <t> [--body <md>]
  mforge epic add-task <rig> --epic <id> --task <id>
  mforge epic assign <rig> --epic <id> [--role <role>]
  mforge epic status <rig> --epic <id>
  mforge epic close <rig> --epic <id>
  mforge epic conflict <rig> --epic <id> --cell <cell> --details <text>
  mforge task split <rig> --task <id> --cells <a,b,c>
  mforge manager tick <rig> [--watch]
  mforge architect docs <rig> --cell <cell> --details <text> [--scope <path>]
  mforge architect contract <rig> --cell <cell> --details <text> [--scope <path>]
  mforge architect design <rig> --cell <cell> --details <text> [--scope <path>]
  mforge report <rig> [--cell <cell>]
  mforge library start <rig> [--addr <addr>]
  mforge library query <rig> --q <query> [--service <name>] [--addr <addr>]
  mforge ssh <rig> --cmd <command...> [--tty]

  # Invoked by Claude Code hooks:
  mforge hook stop [--role <role>]
  mforge hook guardrails

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
	case "request":
		return subcmd.Request(home, rest)
	case "monitor":
		return subcmd.Monitor(home, rest)
	case "epic":
		return subcmd.Epic(home, rest)
	case "architect":
		return subcmd.Architect(home, rest)
	case "report":
		return subcmd.Report(home, rest)
	case "library":
		return subcmd.Library(home, rest)
	case "ssh":
		return subcmd.SSH(home, rest)
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
