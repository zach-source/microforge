package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func Agent(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge agent <spawn|stop|attach|wake|status> ...")
	}
	op := args[0]
	rest := args[1:]
	if op == "status" {
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge agent status <rig> [--cell <cell>] [--role <role>]")
		}
		return agentStatus(home, rest)
	}
	if len(rest) < 3 {
		return fmt.Errorf("usage: mforge agent %s <rig> <cell> <role>", op)
	}
	rigName, cellName, role := rest[0], rest[1], rest[2]
	remote := false
	for i := 3; i < len(rest); i++ {
		if rest[i] == "--remote" {
			remote = true
		}
	}

	cfg, err := store.LoadRigConfig(store.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}

	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil {
		return err
	}
	defer db.Close()
	rigRow, err := store.GetRigByName(db, rigName)
	if err != nil {
		return err
	}
	cellRow, err := store.GetCell(db, rigRow.ID, cellName)
	if err != nil {
		return err
	}
	agentRow, err := store.GetAgentByCellRole(db, cellRow.ID, role)
	if err != nil {
		return err
	}

	session := agentRow.TmuxSession
	worktree := cellRow.WorktreePath

	switch op {
	case "spawn":
		if err := setActiveAgent(worktree, role); err != nil {
			return err
		}
		if _, err := runTmux(cfg, remote, false, "has-session", "-t", session); err == nil {
			fmt.Printf("Session already running: %s\n", session)
			_ = store.UpdateAgentLastSeen(db, agentRow.ID)
			return nil
		}
		cmd, cmdArgs := runtimeForRole(cfg, role)
		remoteWorktree := resolveRemoteWorkdir(cfg, worktree, cellName)
		targs := []string{"new-session", "-d", "-s", session, "-c", remoteWorktree, "--", cmd}
		targs = append(targs, cmdArgs...)
		if _, err := runTmux(cfg, remote, false, targs...); err != nil {
			return err
		}
		_ = store.UpdateAgentLastSeen(db, agentRow.ID)
		fmt.Printf("Spawned %s\n", session)
		return nil

	case "stop":
		if _, err := runTmux(cfg, remote, false, "kill-session", "-t", session); err != nil {
			if isNoSessionErr(err) {
				return nil
			}
			return err
		}
		_ = store.UpdateAgentLastSeen(db, agentRow.ID)
		return nil

	case "attach":
		_, err := runTmux(cfg, remote, true, "attach", "-t", session)
		return err

	case "wake":
		if err := setActiveAgent(worktree, role); err != nil {
			return err
		}
		if _, err := runTmux(cfg, remote, false, "has-session", "-t", session); err != nil {
			return fmt.Errorf("tmux session not running: %s", session)
		}

		prompt := "Check for queued assignments for your role and start working them. If none, respond 'IDLE'."
		if _, err := runTmux(cfg, remote, false, "send-keys", "-t", session, prompt, "Enter"); err != nil {
			return err
		}
		_ = store.UpdateAgentLastSeen(db, agentRow.ID)
		fmt.Printf("Woke %s\n", session)
		return nil

	default:
		return fmt.Errorf("unknown agent subcommand: %s", op)
	}
}

func setActiveAgent(worktree, role string) error {
	idPath := filepath.Join(worktree, ".mf", "active-agent-"+role+".json")
	b, err := os.ReadFile(idPath)
	if err != nil {
		return err
	}
	return util.AtomicWriteFile(filepath.Join(worktree, ".mf", "active-agent.json"), b, 0o644)
}

func isNoSessionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "can't find session") || strings.Contains(msg, "no server running")
}

func agentStatus(home string, args []string) error {
	rigName := args[0]
	var cellName, role string
	remote := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--cell":
			if i+1 < len(args) {
				cellName = args[i+1]
				i++
			}
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--remote":
			remote = true
		}
	}
	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil {
		return err
	}
	defer db.Close()
	cfg, err := store.LoadRigConfig(store.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	rigRow, err := store.GetRigByName(db, rigName)
	if err != nil {
		return err
	}
	agents, err := store.ListAgentsByRig(db, rigRow.ID)
	if err != nil {
		return err
	}
	cells, err := store.ListCellsByRig(db, rigRow.ID)
	if err != nil {
		return err
	}
	cellMap := make(map[string]string, len(cells))
	for _, c := range cells {
		cellMap[c.ID] = c.Name
	}
	for _, a := range agents {
		if cellName != "" && cellMap[a.CellID] != cellName {
			continue
		}
		if role != "" && a.Role != role {
			continue
		}
		state := "stopped"
		if _, err := runTmux(cfg, remote, false, "has-session", "-t", a.TmuxSession); err == nil {
			state = "running"
		}
		lastSeen := ""
		if a.LastSeenAt.Valid {
			lastSeen = a.LastSeenAt.String
		}
		fmt.Printf("%s/%s\t%s\t%s\t%s\n", cellMap[a.CellID], a.Role, a.TmuxSession, state, lastSeen)
	}
	return nil
}

func runtimeForRole(cfg store.RigConfig, role string) (string, []string) {
	cmd := cfg.RuntimeCmd
	args := cfg.RuntimeArgs
	if spec, ok := cfg.RuntimeRoles[role]; ok {
		if strings.TrimSpace(spec.Cmd) != "" {
			cmd = spec.Cmd
		}
		if len(spec.Args) > 0 {
			args = spec.Args
		}
	}
	return cmd, args
}

func resolveRemoteWorkdir(cfg store.RigConfig, localWorktree, cellName string) string {
	if strings.TrimSpace(cfg.RemoteWorkdir) == "" {
		return localWorktree
	}
	out := strings.ReplaceAll(cfg.RemoteWorkdir, "{cell}", cellName)
	return out
}

func runTmux(cfg store.RigConfig, remote bool, tty bool, args ...string) (util.CmdResult, error) {
	if remote || strings.TrimSpace(cfg.RemoteHost) != "" {
		if strings.TrimSpace(cfg.RemoteHost) == "" {
			return util.CmdResult{}, fmt.Errorf("remote_host is not configured")
		}
		cmd, sshArgs := buildSSHCommand(cfg, append([]string{"tmux"}, args...), tty)
		return util.Run(nil, cmd, sshArgs...)
	}
	return util.Run(nil, "tmux", args...)
}
