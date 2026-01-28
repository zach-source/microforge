package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/example/microforge/internal/context"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/subcmd"
)

func usage() string {
	return strings.TrimSpace(`
mforge — Microforge (Beads + tmux + Claude Code hooks)

Usage:
  mforge init <rig> --repo <path>

  mforge cell add <cell> --scope <path-prefix>
  mforge cell bootstrap <cell> [--architect] [--single]

  mforge agent spawn <cell> <role>
  mforge agent stop  <cell> <role>
  mforge agent exit  <cell> <role>
  mforge agent attach <cell> <role>
  mforge agent wake <cell> <role>
  mforge agent relaunch <cell> <role>
  mforge agent restart <cell> <role>
  mforge agent send <cell> <role> <message> [--no-enter]
  mforge agent logs <cell> <role> [--follow] [--lines <n>] [--all]
  mforge agent heartbeat <cell> <role>
  mforge agent create <path> --description <text> [--class crew|worker]
  mforge agent bootstrap <name>
  mforge agent status [--cell <cell>] [--role <role>] [--remote] [--json]
  mforge status [--cell <cell>] [--role <role>] [--json]

  mforge task create --title <t> [--body <md>] [--scope <path-prefix>] [--kind improve|fix|review|monitor|doc]
  mforge task update --task <id> --scope <path-prefix>
  mforge task complete --task <id> [--reason <text>] [--force]
  mforge task delete --task <id> [--reason <text>] [--force] [--cascade] [--hard] [--dry-run]
  mforge task list
  mforge task decompose --task <id> --titles <a,b,c> [--kind <kind>]
  mforge scope list
  mforge scope show --scope <path-prefix>
  mforge engine run [--wait]
  mforge engine emit --type <event> [--scope <path>] [--title <text>] [--source <role>] [--payload <json>]
  mforge engine drain [--keep]
  mforge convoy start --epic <id> [--role <role>] [--title <text>]

  mforge assign --task <id> --cell <cell> --role builder|monitor|reviewer|architect|cell [--promise <token>] [--quick]
  mforge quick-assign <bead-id> <cell> [--role <role>] [--promise <token>]
  mforge quick-assign <bead-id> <cell> [--role <role>] [--promise <token>]
  mforge request create --cell <cell> --role <role> --severity <sev> --priority <p> --scope <path> --payload <json>
  mforge request list [--cell <cell>] [--status <status>] [--priority <p>]
  mforge request triage --request <id> --action create-task|merge|block
  mforge monitor run-tests <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>]
  mforge monitor run <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>] [--observation <text>]
  mforge epic create --title <t> [--body <md>]
  mforge epic create --title <t> --short-id <id>
  mforge epic design <id|short-id>
  mforge epic tree <id|short-id>
  mforge epic add-task --epic <id> --task <id>
  mforge epic assign --epic <id> [--role <role>]
  mforge epic status --epic <id>
  mforge epic close --epic <id>
  mforge epic conflict --epic <id> --cell <cell> --details <text>
  mforge task split --task <id> --cells <a,b,c>
  mforge manager tick [--watch] [--stop-idle]
  mforge manager assign [--role <role>]
  mforge turn start [--name <name>]
  mforge turn status
  mforge turn end [--report]
  mforge turn slate
  mforge turn list
  mforge turn diff [--id <turn>]
  mforge turn run [--role <role>] [--wait]
  mforge round start [--wait]
  mforge round review [--wait] [--all] [--changes-only] [--base <branch>]
  mforge round merge --feature <branch> [--base <branch>]
  mforge checkpoint [--message <text>]
  mforge wait [--turn <id>] [--interval <seconds>]
mforge bead create --type <type> --title <title> [--cell <cell>] [--turn <id>] ...
mforge bead list [--type <type>] [--status <status>] [--cell <cell>] [--turn <id>]
mforge bead show <id>
mforge bead close <id>
mforge bead status <id> <status>
mforge bead triage --id <id> --cell <cell> --role <role>
  mforge bead dep add <id> <dep>
  mforge bead template --type <type> --title <title> --cell <cell>
  mforge review create --title <title> --cell <cell>
  mforge pr create --title <title> --cell <cell> [--url <url>]
  mforge pr ready <id>
  mforge pr link-review <pr_id> <review_id>
  mforge merge run --as merge-manager [--turn <id>] [--dry-run]
  mforge coordinator sync [--turn <id>]
  mforge digest render [--turn <id>]
  mforge build record --service <name> --image <tag>
  mforge deploy record --env <env> --service <name>
  mforge contract create --title <title> --cell <cell> --scope <path>
  mforge architect docs --cell <cell> --details <text> [--scope <path>]
  mforge architect contract --cell <cell> --details <text> [--scope <path>]
  mforge architect design --cell <cell> --details <text> [--scope <path>]
  mforge report [--cell <cell>]
  mforge library start [--addr <addr>]
  mforge library query --q <query> [--service <name>] [--addr <addr>]
  mforge watch [--interval <seconds>] [--role <role>] [--fswatch] [--tui]
  mforge tui [--interval <seconds>] [--remote] [--watch] [--role <role>]
  mforge migrate beads [--all]
  mforge migrate rig [--all]
  mforge rig <list|delete|rename|backup|restore> ...
  mforge ssh <rig> --cmd <command...> [--tty]
  mforge context <get|set|unset|list> [<rig>]
  mforge completions <install|path|bash|zsh>

  # Invoked by Claude Code hooks:
  mforge hook stop [--role <role>]
  mforge hook guardrails
  mforge hook emit --event <name>

Environment:
  MF_HOME   override default home (~/.microforge)
`)
}

func Run(args []string) error {
	if len(args) == 0 {
		fmt.Println(usage())
		return nil
	}
	home := rig.DefaultHome()
	if v := strings.TrimSpace(os.Getenv("MF_HOME")); v != "" {
		home = v
	}

	cmd := args[0]
	rest := args[1:]
	activeRig := ""
	if cmd != "context" {
		if s, err := context.Load(home); err == nil {
			activeRig = strings.TrimSpace(s.ActiveRig)
		}
	}
	if requiresActiveRig(cmd) && strings.TrimSpace(activeRig) == "" {
		return fmt.Errorf("no active rig set; run `mforge context set <rig>`")
	}
	rest = maybeInjectActiveRig(cmd, rest, activeRig)

	switch cmd {
	case "help", "-h", "--help":
		if len(rest) > 0 {
			if msg, ok := commandUsage(rest[0]); ok {
				fmt.Println(msg)
				return nil
			}
		}
		fmt.Println(usage())
		return nil
	case "init":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Init(home, rest)
	case "cell":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Cell(home, rest)
	case "agent":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Agent(home, rest)
	case "status":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Status(home, rest)
	case "task":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Task(home, rest)
	case "scope":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Scope(home, rest)
	case "engine":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Engine(home, rest)
	case "convoy":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Convoy(home, rest)
	case "request":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Request(home, rest)
	case "monitor":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Monitor(home, rest)
	case "epic":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Epic(home, rest)
	case "architect":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Architect(home, rest)
	case "report":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Report(home, rest)
	case "library":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Library(home, rest)
	case "watch":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Watch(home, rest)
	case "migrate":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Migrate(home, rest)
	case "tui":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.TUI(home, rest)
	case "rig":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Rig(home, rest)
	case "context":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Context(home, rest)
	case "completions":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Completions(home, rest)
	case "ssh":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.SSH(home, rest)
	case "assign":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Assign(home, rest)
	case "quick-assign":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.QuickAssign(home, rest)
	case "manager":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Manager(home, rest)
	case "turn":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		if len(rest) == 0 {
			rest = []string{"status"}
		}
		if len(rest) > 0 && rest[0] == "run" {
			return subcmd.TurnRun(home, rest[1:])
		}
		return subcmd.Turn(home, rest)
	case "round":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Round(home, rest)
	case "bead":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		if len(rest) > 0 && rest[0] == "template" {
			return subcmd.BeadTemplate(home, rest[1:])
		}
		return subcmd.Bead(home, rest)
	case "review":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Review(home, rest)
	case "pr":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.PR(home, rest)
	case "merge":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Merge(home, rest)
	case "wait":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Wait(home, rest)
	case "checkpoint":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Checkpoint(home, rest)
	case "coordinator":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Coordinator(home, rest)
	case "digest":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Digest(home, rest)
	case "build":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Build(home, rest)
	case "deploy":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Deploy(home, rest)
	case "contract":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Contract(home, rest)
	case "hook":
		if hasHelpFlag(rest) {
			printCommandUsage(cmd)
			return nil
		}
		return subcmd.Hook(home, rest)
	default:
		return errors.New("unknown command: " + cmd + "\n\n" + usage())
	}
}

func maybeInjectActiveRig(cmd string, rest []string, activeRig string) []string {
	if strings.TrimSpace(activeRig) == "" {
		return rest
	}
	switch cmd {
	case "cell", "task", "request", "epic", "manager", "turn", "bead", "review", "pr", "merge", "coordinator", "digest", "build", "deploy", "contract", "architect", "library", "engine", "convoy", "scope", "monitor", "round", "migrate", "watch":
		return injectAfterSubcommand(rest, activeRig)
	case "agent":
		return injectAfterSubcommandWithOverride(rest, activeRig, map[string]bool{"create": true, "bootstrap": true})
	case "assign", "quick-assign", "wait", "report", "ssh", "checkpoint", "tui", "status":
		return injectAtStart(rest, activeRig)
	default:
		return rest
	}
}

func injectAfterSubcommandWithOverride(rest []string, activeRig string, subcommands map[string]bool) []string {
	if len(rest) == 0 {
		return rest
	}
	if subcommands[rest[0]] {
		out := make([]string, 0, len(rest)+1)
		out = append(out, rest[0], activeRig)
		out = append(out, rest[1:]...)
		return out
	}
	return injectAfterSubcommand(rest, activeRig)
}

func requiresActiveRig(cmd string) bool {
	switch cmd {
	case "cell", "agent", "task", "request", "epic", "manager", "turn", "bead", "review", "pr", "merge", "coordinator", "digest", "build", "deploy", "contract", "architect", "library", "engine", "convoy", "scope", "monitor", "assign", "quick-assign", "wait", "report", "ssh", "round", "checkpoint", "tui", "watch", "status":
		return true
	default:
		return false
	}
}

func injectAfterSubcommand(rest []string, activeRig string) []string {
	if len(rest) == 0 {
		return rest
	}
	if len(rest) == 1 {
		return []string{rest[0], activeRig}
	}
	if rest[1] == activeRig {
		return rest
	}
	out := make([]string, 0, len(rest)+1)
	out = append(out, rest[0], activeRig)
	out = append(out, rest[1:]...)
	return out
}

func injectAtStart(rest []string, activeRig string) []string {
	if len(rest) == 0 || strings.HasPrefix(rest[0], "-") {
		return append([]string{activeRig}, rest...)
	}
	return rest
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "help", "-h", "--help":
			return true
		}
	}
	return false
}

func printCommandUsage(cmd string) {
	if msg, ok := commandUsage(cmd); ok {
		fmt.Println(msg)
	}
}

func commandUsage(cmd string) (string, bool) {
	switch cmd {
	case "init":
		return "mforge init <rig> --repo <path>", true
	case "cell":
		return strings.TrimSpace(`
mforge cell add <cell> --scope <path-prefix>
mforge cell bootstrap <cell> [--architect] [--single]
mforge cell agent-file <cell> --role <role>
`), true
	case "agent":
		return strings.TrimSpace(`
mforge agent spawn <cell> <role>
mforge agent stop <cell> <role>
mforge agent exit <cell> <role>
mforge agent attach <cell> <role>
mforge agent wake <cell> <role>
mforge agent relaunch <cell> <role>
mforge agent restart <cell> <role>
mforge agent send <cell> <role> <message> [--no-enter]
mforge agent logs <cell> <role> [--follow] [--lines <n>] [--all]
mforge agent heartbeat <cell> <role>
mforge agent create <path> --description <text> [--class crew|worker]
mforge agent bootstrap <name>
mforge agent status [--cell <cell>] [--role <role>] [--remote] [--json]
`), true
	case "status":
		return "mforge status [--cell <cell>] [--role <role>] [--json]", true
	case "task":
		return strings.TrimSpace(`
mforge task create --title <t> [--body <md>] [--scope <path-prefix>] [--kind improve|fix|review|monitor|doc]
mforge task update --task <id> --scope <path-prefix>
mforge task complete --task <id> [--reason <text>] [--force]
mforge task delete --task <id> [--reason <text>] [--force] [--cascade] [--hard] [--dry-run]
mforge task list
mforge task split --task <id> --cells <a,b,c>
mforge task decompose --task <id> --titles <a,b,c> [--kind <kind>]
`), true
	case "scope":
		return strings.TrimSpace(`
mforge scope list
mforge scope show --scope <path-prefix>
`), true
	case "engine":
		return strings.TrimSpace(`
mforge engine run [--wait]
mforge engine run --rounds <n>
mforge engine run --completion-promise <text>
mforge engine emit --type <event> [--scope <path>] [--title <text>] [--source <role>] [--payload <json>]
mforge engine drain [--keep]
`), true
	case "convoy":
		return "mforge convoy start --epic <id> [--role <role>] [--title <text>]", true
	case "assign":
		return "mforge assign --task <id> --cell <cell> --role <role> [--promise <token>] [--quick]", true
	case "quick-assign":
		return "mforge quick-assign <bead-id> <cell> [--role <role>] [--promise <token>]", true
	case "request":
		return strings.TrimSpace(`
mforge request create --cell <cell> --role <role> --severity <sev> --priority <p> --scope <path> --payload <json>
mforge request list [--cell <cell>] [--status <status>] [--priority <p>]
mforge request triage --request <id> --action create-task|merge|block
`), true
	case "monitor":
		return strings.TrimSpace(`
mforge monitor run-tests <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>]
mforge monitor run <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>] [--observation <text>]
`), true
	case "epic":
		return strings.TrimSpace(`
mforge epic create --title <t> [--body <md>]
mforge epic create --title <t> --short-id <id>
mforge epic design <id|short-id>
mforge epic tree <id|short-id>
mforge epic add-task --epic <id> --task <id>
mforge epic assign --epic <id> [--role <role>]
mforge epic status --epic <id>
mforge epic close --epic <id>
mforge epic conflict --epic <id> --cell <cell> --details <text>
`), true
	case "manager":
		return strings.TrimSpace(`
  mforge manager tick [--watch] [--stop-idle]
mforge manager assign [--role <role>]
`), true
	case "turn":
		return strings.TrimSpace(`
mforge turn start
mforge turn status
mforge turn end
mforge turn slate
mforge turn list
mforge turn diff [--id <turn>]
mforge turn run [--role <role>] [--wait]
`), true
	case "round":
		return strings.TrimSpace(`
mforge round start [--wait]
mforge round review [--wait] [--all] [--changes-only] [--base <branch>]
mforge round merge --feature <branch> [--base <branch>]
`), true
	case "bead":
		return strings.TrimSpace(`
mforge bead create --type <type> --title <title> [--priority <p>] [--status <status>] [--cell <cell>] [--role <role>] [--scope <path>] [--turn <id>] [--severity <sev>] [--description <text>] [--acceptance <text>] [--compat <text>] [--links <text>] [--deps <a,b,c>]
mforge bead list [--type <type>] [--status <status>] [--cell <cell>] [--priority <p>] [--turn <id>]
mforge bead show <id>
mforge bead close <id> [--reason <text>]
mforge bead status <id> <status> [--reason <text>]
mforge bead triage --id <id> --cell <cell> --role <role> [--turn <id>] [--promise <token>]
mforge bead dep add <id> <dep>
mforge bead template --type <type> --title <title> --cell <cell> [--scope <path>] [--priority <p>] [--turn <id>]
`), true
	case "review":
		return "mforge review create --title <title> --cell <cell> [--scope <path>] [--turn <id>]", true
	case "pr":
		return strings.TrimSpace(`
mforge pr create --title <title> --cell <cell> [--url <url>] [--turn <id>] [--status <status>]
mforge pr ready <id>
mforge pr link-review <pr_id> <review_id>
`), true
	case "merge":
		return "mforge merge run [--turn <id>] [--as merge-manager] [--dry-run]", true
	case "coordinator":
		return "mforge coordinator sync [--turn <id>]", true
	case "digest":
		return "mforge digest render [--turn <id>]", true
	case "build":
		return "mforge build record --service <name> --image <tag> [--status <status>] [--turn <id>]", true
	case "deploy":
		return "mforge deploy record --env <env> --service <name> [--status <status>] [--turn <id>]", true
	case "contract":
		return "mforge contract create --title <title> --cell <cell> --scope <path> [--acceptance <text>] [--compat <text>] [--links <text>]", true
	case "architect":
		return strings.TrimSpace(`
mforge architect docs --cell <cell> --details <text> [--scope <path>]
mforge architect contract --cell <cell> --details <text> [--scope <path>]
mforge architect design --cell <cell> --details <text> [--scope <path>]
`), true
	case "report":
		return "mforge report [--cell <cell>]", true
	case "library":
		return strings.TrimSpace(`
mforge library start [--addr <addr>]
mforge library query --q <query> [--service <name>] [--addr <addr>]
`), true
	case "watch":
		return "mforge watch [--interval <seconds>] [--role <role>] [--fswatch] [--tui]", true
	case "migrate":
		return "mforge migrate beads [--all]\nmforge migrate rig [--all]", true
	case "tui":
		return "mforge tui [--interval <seconds>] [--remote] [--watch] [--role <role>]", true
	case "context":
		return strings.TrimSpace(`
mforge context get
mforge context set <rig>
mforge context unset
mforge context list
`), true
	case "rig":
		return strings.TrimSpace(`
mforge rig list
mforge rig delete <rig>
mforge rig rename <old> <new>
mforge rig backup <rig> [--out <path>]
mforge rig restore <archive> --name <rig> [--force]
`), true
	case "completions":
		return strings.TrimSpace(`
mforge completions install
mforge completions path
mforge completions bash
mforge completions zsh
`), true
	case "ssh":
		return "mforge ssh --cmd <command...> [--tty]", true
	case "wait":
		return "mforge wait [--turn <id>] [--interval <seconds>]", true
	case "checkpoint":
		return "mforge checkpoint [--message <text>]", true
	case "hook":
		return strings.TrimSpace(`
mforge hook stop [--role <role>]
mforge hook guardrails
mforge hook emit --event <name>
`), true
	default:
		return "", false
	}
}
