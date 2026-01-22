package subcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func Cell(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mf cell <add|bootstrap> ...")
	}
	op := args[0]
	rest := args[1:]
	switch op {
	case "add":
		if len(rest) < 2 { return fmt.Errorf("usage: mf cell add <rig> <cell> --scope <path-prefix>") }
		rigName, cellName := rest[0], rest[1]
		scope := ""
		for i := 2; i < len(rest); i++ {
			if rest[i] == "--scope" && i+1 < len(rest) { scope = rest[i+1]; i++ }
		}
		if strings.TrimSpace(scope) == "" { return fmt.Errorf("--scope is required") }
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil { return err }
		defer db.Close()
		rigRow, err := store.GetRigByName(db, rigName)
		if err != nil { return err }
		worktree := store.CellWorktreeDir(home, rigName, cellName)
		_ = util.EnsureDir(filepath.Dir(worktree))
		if _, err := store.CreateCell(db, rigRow.ID, cellName, scope, worktree); err != nil { return err }
		fmt.Printf("Created cell %q (scope=%q)\n", cellName, scope)
		fmt.Printf("Worktree path: %s\n", worktree)
		return nil

	case "bootstrap":
		if len(rest) != 2 { return fmt.Errorf("usage: mf cell bootstrap <rig> <cell>") }
		rigName, cellName := rest[0], rest[1]
		cfg, err := store.LoadRigConfig(store.RigConfigPath(home, rigName))
		if err != nil { return err }
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil { return err }
		defer db.Close()
		rigRow, err := store.GetRigByName(db, rigName)
		if err != nil { return err }
		cellRow, err := store.GetCell(db, rigRow.ID, cellName)
		if err != nil { return err }

		wt := cellRow.WorktreePath
		if _, err := os.Stat(wt); err != nil {
			if _, err := os.Stat(filepath.Join(cfg.RepoPath, ".git")); err == nil {
				branch := fmt.Sprintf("cell/%s/%s", cellName, strings.ReplaceAll(store.MustNow(), ":", ""))
				if _, err := util.Run(nil, "git", "-C", cfg.RepoPath, "worktree", "add", "-b", branch, wt); err != nil {
					return fmt.Errorf("git worktree add failed: %w", err)
				}
			} else {
				if err := util.EnsureDir(wt); err != nil { return err }
			}
		}
		_ = util.EnsureDir(filepath.Join(wt, ".claude"))
		_ = util.EnsureDir(filepath.Join(wt, ".mf"))
		for _, p := range []string{"mail/inbox","mail/outbox","mail/archive"} { _ = util.EnsureDir(filepath.Join(wt, p)) }

		for _, role := range []string{"builder","monitor","reviewer"} {
			tmuxSession := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, cellName, role)
			agentName := fmt.Sprintf("%s-%s", cellName, role)
			agentRow, err := store.CreateAgent(db, rigRow.ID, cellRow.ID, role, agentName, tmuxSession)
			if err != nil { return err }
			identity := map[string]any{
				"rig_name": rigName,
				"db_path": store.DBPath(home, rigName),
				"cell_name": cellName,
				"role": role,
				"agent_id": agentRow.ID,
				"scope": cellRow.ScopePrefix,
				"worktree_path": wt,
				"inbox":"mail/inbox","outbox":"mail/outbox","archive":"mail/archive",
			}
			b, _ := json.MarshalIndent(identity, "", "  ")
			metaDir := store.CellMetaDir(home, rigName, cellName)
			_ = util.EnsureDir(metaDir)
			if err := util.AtomicWriteFile(store.CellRoleMetaPath(home, rigName, cellName, role), b, 0o644); err != nil { return err }
			if err := util.AtomicWriteFile(filepath.Join(wt, ".mf", "active-agent-"+role+".json"), b, 0o644); err != nil { return err }
		}

		settings := `{
  "hooks": {
    "Stop": [
      { "hooks": [ { "type": "command", "command": "mf hook stop --role builder" } ] },
      { "hooks": [ { "type": "command", "command": "mf hook stop --role monitor" } ] },
      { "hooks": [ { "type": "command", "command": "mf hook stop --role reviewer" } ] }
    ],
    "PreToolUse": [
      { "matcher": "Write|Edit", "hooks": [ { "type": "command", "command": "mf hook guardrails" } ] }
    ],
    "PermissionRequest": [
      { "matcher": "Bash(*)", "hooks": [ { "type": "command", "command": "mf hook guardrails" } ] }
    ]
  }
}`
		if err := util.AtomicWriteFile(store.CellClaudeSettingsPath(home, rigName, cellName), []byte(settings+"\n"), 0o644); err != nil { return err }

		// default active agent pointer for hooks (builder)
		b0, _ := os.ReadFile(filepath.Join(wt, ".mf", "active-agent-builder.json"))
		_ = util.AtomicWriteFile(filepath.Join(wt, ".mf", "active-agent.json"), b0, 0o644)

		fmt.Printf("Bootstrapped cell %q at %s\n", cellName, wt)
		fmt.Println("Created agents: builder, monitor, reviewer")
		return nil

	default:
		return fmt.Errorf("unknown cell subcommand: %s", op)
	}
}
