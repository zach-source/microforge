package subcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func Agent(home string, args []string) error {
	if len(args) < 1 { return fmt.Errorf("usage: mf agent <spawn|stop|attach|wake> ...") }
	op := args[0]
	rest := args[1:]
	if len(rest) < 3 { return fmt.Errorf("usage: mf agent %s <rig> <cell> <role>", op) }
	rigName, cellName, role := rest[0], rest[1], rest[2]

	cfg, err := store.LoadRigConfig(store.RigConfigPath(home, rigName))
	if err != nil { return err }

	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil { return err }
	defer db.Close()
	rigRow, err := store.GetRigByName(db, rigName)
	if err != nil { return err }
	cellRow, err := store.GetCell(db, rigRow.ID, cellName)
	if err != nil { return err }
	agentRow, err := store.GetAgentByCellRole(db, cellRow.ID, role)
	if err != nil { return err }

	session := agentRow.TmuxSession
	worktree := cellRow.WorktreePath

	switch op {
	case "spawn":
		idPath := filepath.Join(worktree, ".mf", "active-agent-"+role+".json")
		b, err := os.ReadFile(idPath)
		if err != nil { return err }
		if err := util.AtomicWriteFile(filepath.Join(worktree, ".mf", "active-agent.json"), b, 0o644); err != nil { return err }

		if _, err := util.Run(nil, "tmux", "has-session", "-t", session); err == nil {
			return fmt.Errorf("tmux session already exists: %s", session)
		}
		targs := []string{"new-session","-d","-s",session,"-c",worktree,"--",cfg.RuntimeCmd}
		targs = append(targs, cfg.RuntimeArgs...)
		if _, err := util.Run(nil, "tmux", targs...); err != nil { return err }
		fmt.Printf("Spawned %s\n", session)
		return nil

	case "stop":
		_, err := util.Run(nil, "tmux", "kill-session", "-t", session)
		return err

	case "attach":
		_, err := util.Run(nil, "tmux", "attach", "-t", session)
		return err

	case "wake":
		idPath := filepath.Join(worktree, ".mf", "active-agent-"+role+".json")
		b, err := os.ReadFile(idPath)
		if err != nil { return err }
		if err := util.AtomicWriteFile(filepath.Join(worktree, ".mf", "active-agent.json"), b, 0o644); err != nil { return err }

		prompt := "Check for queued assignments for your role and start working them. If none, respond 'IDLE'."
		if _, err := util.Run(nil, "tmux", "send-keys", "-t", session, prompt, "Enter"); err != nil { return err }
		fmt.Printf("Woke %s\n", session)
		return nil

	default:
		return fmt.Errorf("unknown agent subcommand: %s", op)
	}
}
