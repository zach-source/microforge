package subcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
	"github.com/example/microforge/internal/util"
)

func Cell(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge cell <add|bootstrap|agent-file> ...")
	}
	op := args[0]
	rest := args[1:]
	switch op {
	case "add":
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge cell add <rig> <cell> --scope <path-prefix>")
		}
		rigName, cellName := rest[0], rest[1]
		scope := ""
		for i := 2; i < len(rest); i++ {
			if rest[i] == "--scope" && i+1 < len(rest) {
				scope = rest[i+1]
				i++
			}
		}
		if strings.TrimSpace(scope) == "" {
			return fmt.Errorf("--scope is required")
		}
		worktree := rig.CellWorktreeDir(home, rigName, cellName)
		_ = util.EnsureDir(filepath.Dir(worktree))
		cellCfg := rig.CellConfig{
			Name:         cellName,
			ScopePrefix:  scope,
			WorktreePath: worktree,
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		}
		if err := util.EnsureDir(rig.CellDir(home, rigName, cellName)); err != nil {
			return err
		}
		if err := rig.SaveCellConfig(rig.CellConfigPath(home, rigName, cellName), cellCfg); err != nil {
			return err
		}
		fmt.Printf("Created cell %q (scope=%q)\n", cellName, scope)
		fmt.Printf("Worktree path: %s\n", worktree)
		return nil

	case "bootstrap":
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge cell bootstrap <rig> <cell> [--architect] [--single]")
		}
		rigName, cellName := rest[0], rest[1]
		withArchitect := false
		single := false
		for i := 2; i < len(rest); i++ {
			if rest[i] == "--architect" {
				withArchitect = true
			}
			if rest[i] == "--single" {
				single = true
			}
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
		if err != nil {
			return err
		}

		wt := cellCfg.WorktreePath
		repoHasGit := false
		if _, err := os.Stat(filepath.Join(cfg.RepoPath, ".git")); err == nil {
			repoHasGit = true
		}
		if _, err := os.Stat(wt); err != nil {
			if repoHasGit {
				branch := cellBranchName(home, rigName, cellName)
				if _, err := util.Run(nil, "git", "-C", cfg.RepoPath, "worktree", "add", "-b", branch, wt); err != nil {
					return fmt.Errorf("git worktree add failed: %w", err)
				}
			} else {
				if err := util.EnsureDir(wt); err != nil {
					return err
				}
			}
		} else if repoHasGit {
			if _, err := os.Stat(filepath.Join(wt, ".git")); err != nil {
				if isDirEmpty(wt) {
					branch := cellBranchName(home, rigName, cellName)
					if _, err := util.Run(nil, "git", "-C", cfg.RepoPath, "worktree", "add", "-b", branch, wt); err != nil {
						return fmt.Errorf("git worktree add failed: %w", err)
					}
				} else {
					fmt.Printf("Warning: worktree %s is not a git worktree (non-empty directory)\n", wt)
				}
			}
		}
		_ = util.EnsureDir(filepath.Join(wt, ".claude"))
		_ = util.EnsureDir(filepath.Join(wt, ".mf"))
		if err := ensureHookConfig(wt); err != nil {
			return err
		}
		for _, p := range []string{"mail/inbox", "mail/outbox", "mail/archive"} {
			_ = util.EnsureDir(filepath.Join(wt, p))
		}
		ensureClaudeSymlink(cfg.RepoPath, wt)

		roles := []string{"builder", "monitor", "reviewer"}
		if single {
			roles = []string{"cell"}
		}
		if withArchitect {
			roles = append(roles, "architect")
		}
		writeRoleGuides(wt, single, withArchitect)

		client := beads.Client{RepoPath: cfg.RepoPath}
		issues, _ := client.List(nil)
		agentID := ""
		agentClass := "worker"
		if spec, err := loadAgentSpec(cfg.RepoPath, cellName); err == nil {
			agentID = strings.TrimSpace(spec.BeadID)
			if strings.TrimSpace(spec.Class) != "" {
				agentClass = spec.Class
			}
		}
		for _, role := range roles {
			tmuxSession := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, cellName, role)
			guide := readRoleGuide(wt, role)
			roleIssue, updated, err := ensureRoleBead(issues, client, cellName, role, cellCfg.ScopePrefix, guide)
			if err != nil {
				return err
			}
			issues = updated
			mailIssue, updated, err := ensureMailboxBead(issues, client, cellName, role, "mail/inbox", "mail/outbox", wt)
			if err != nil {
				return err
			}
			issues = updated
			hookIssue, updated, err := ensureHookBead(issues, client, cellName, role, cellCfg.ScopePrefix)
			if err != nil {
				return err
			}
			issues = updated

			identity := map[string]any{
				"rig_name":      rigName,
				"rig_home":      home,
				"repo_path":     cfg.RepoPath,
				"cell_name":     cellName,
				"role":          role,
				"scope":         cellCfg.ScopePrefix,
				"worktree_path": wt,
				"tmux_session":  tmuxSession,
				"inbox":         "mail/inbox",
				"outbox":        "mail/outbox",
				"archive":       "mail/archive",
				"agent_id":      agentID,
				"role_id":       roleIssue.ID,
				"mailbox_id":    mailIssue.ID,
				"hook_id":       hookIssue.ID,
				"class":         agentClass,
			}
			b, _ := json.MarshalIndent(identity, "", "  ")
			metaDir := rig.CellMetaDir(home, rigName, cellName)
			_ = util.EnsureDir(metaDir)
			if err := util.AtomicWriteFile(rig.CellRoleMetaPath(home, rigName, cellName, role), b, 0o644); err != nil {
				return err
			}
			if err := util.AtomicWriteFile(filepath.Join(wt, ".mf", "active-agent-"+role+".json"), b, 0o644); err != nil {
				return err
			}
		}

		var stopHooks []string
		for _, role := range roles {
			stopHooks = append(stopHooks, fmt.Sprintf(`      { "hooks": [ { "type": "command", "command": "mforge hook stop --role %s" } ] }`, role))
		}
		for i := range stopHooks {
			stopHooks[i] = strings.TrimSuffix(stopHooks[i], " } ] }") + ` }, { "type": "command", "command": "mforge hook emit --event claude_stop" } ] }`
		}
		settings := fmt.Sprintf(`{
  "permissions": { "allow": ["Bash", "Read", "Write", "Edit"] },
  "hooks": {
    "Stop": [
%s
    ],
    "PreToolUse": [
      { "matcher": "Write|Edit", "hooks": [ { "type": "command", "command": "mforge hook guardrails" }, { "type": "command", "command": "mforge hook emit --event claude_pre_tool" } ] }
    ],
    "PermissionRequest": [
      { "matcher": "Bash", "hooks": [ { "type": "command", "command": "mforge hook guardrails" }, { "type": "command", "command": "mforge hook emit --event claude_permission" } ] }
    ]
  }
}`, strings.Join(stopHooks, ",\n"))
		if err := util.AtomicWriteFile(rig.CellClaudeSettingsPath(home, rigName, cellName), []byte(settings+"\n"), 0o644); err != nil {
			return err
		}

		defaultRole := "builder"
		if single {
			defaultRole = "cell"
		}
		// default active agent pointer for hooks
		b0, _ := os.ReadFile(filepath.Join(wt, ".mf", "active-agent-"+defaultRole+".json"))
		_ = util.AtomicWriteFile(filepath.Join(wt, ".mf", "active-agent.json"), b0, 0o644)

		copyKubeconfig(wt)

		fmt.Printf("Bootstrapped cell %q at %s\n", cellName, wt)
		switch {
		case single && withArchitect:
			fmt.Println("Created agents: cell, architect")
		case single:
			fmt.Println("Created agents: cell")
		case withArchitect:
			fmt.Println("Created agents: builder, monitor, reviewer, architect")
		default:
			fmt.Println("Created agents: builder, monitor, reviewer")
		}
		return nil

	case "agent-file":
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge cell agent-file <rig> <cell> --role <role>")
		}
		rigName, cellName := rest[0], rest[1]
		role := ""
		for i := 2; i < len(rest); i++ {
			if rest[i] == "--role" && i+1 < len(rest) {
				role = rest[i+1]
				i++
			}
		}
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("--role is required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
		if err != nil {
			return err
		}
		tmuxSession := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, cellName, role)
		identity := map[string]any{
			"rig_name":      rigName,
			"rig_home":      home,
			"repo_path":     cfg.RepoPath,
			"cell_name":     cellName,
			"role":          role,
			"scope":         cellCfg.ScopePrefix,
			"worktree_path": cellCfg.WorktreePath,
			"tmux_session":  tmuxSession,
			"inbox":         "mail/inbox",
			"outbox":        "mail/outbox",
			"archive":       "mail/archive",
		}
		b, _ := json.MarshalIndent(identity, "", "  ")
		metaDir := rig.CellMetaDir(home, rigName, cellName)
		_ = util.EnsureDir(metaDir)
		if err := util.AtomicWriteFile(rig.CellRoleMetaPath(home, rigName, cellName, role), b, 0o644); err != nil {
			return err
		}
		if err := util.AtomicWriteFile(filepath.Join(cellCfg.WorktreePath, ".mf", "active-agent-"+role+".json"), b, 0o644); err != nil {
			return err
		}
		fmt.Printf("Wrote agent file for %s/%s (%s)\n", cellName, role, rig.CellRoleMetaPath(home, rigName, cellName, role))
		return nil

	default:
		return fmt.Errorf("unknown cell subcommand: %s", op)
	}
}

func copyKubeconfig(worktree string) {
	src := strings.TrimSpace(os.Getenv("MF_KUBECONFIG"))
	if src == "" {
		src = strings.TrimSpace(os.Getenv("KUBECONFIG"))
	}
	if src == "" {
		return
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return
	}
	dst := filepath.Join(worktree, "kubeconfig.yaml")
	_ = util.AtomicWriteFile(dst, b, 0o600)
}

func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

func cellBranchName(home, rigName, cellName string) string {
	turnID := ""
	if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
		turnID = strings.TrimSpace(state.ID)
	}
	epic := strings.TrimSpace(os.Getenv("MF_EPIC"))
	parts := []string{"cell", cellName}
	if turnID != "" {
		parts = append(parts, turnID)
	}
	if epic != "" {
		parts = append(parts, epic)
	}
	if turnID == "" && epic == "" {
		parts = append(parts, strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339), ":", ""))
	}
	return strings.Join(parts, "/")
}

func writeRoleGuides(worktree string, single bool, withArchitect bool) {
	dir := filepath.Join(worktree, ".mf", "roles")
	_ = util.EnsureDir(dir)
	roles := []string{"builder", "monitor", "reviewer"}
	if single {
		roles = []string{"cell"}
	}
	if withArchitect {
		roles = append(roles, "architect")
	}
	for _, role := range roles {
		path := filepath.Join(dir, role+".md")
		if _, err := os.Stat(path); err == nil {
			continue
		}
		content := defaultRoleGuide(role)
		if strings.TrimSpace(content) == "" {
			continue
		}
		_ = util.AtomicWriteFile(path, []byte(content+"\n"), 0o644)
	}
}

func readRoleGuide(worktree, role string) string {
	path := filepath.Join(worktree, ".mf", "roles", role+".md")
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func defaultRoleGuide(role string) string {
	switch role {
	case "builder":
		return "Builder role\n\n- Implement assigned tasks strictly in-scope.\n- Update tests for changes.\n- If out-of-scope work is needed, emit a request bead and stop.\n- When complete, include the promise token in the outbox message."
	case "reviewer":
		return "Reviewer role\n\n- Validate acceptance criteria and scope boundaries.\n- Reject shortcuts (no silent contract changes, no skipped tests).\n- Emit ReviewRequest for missing tests or defects.\n- Approve only when changes are correct and safe."
	case "monitor":
		return "Monitor role\n\n- Run scoped tests and checks.\n- Only emit events/requests; do not modify code.\n- If flakiness or regression is detected, emit MonitorRequest.\n- Report signals with scope and impact."
	case "architect":
		return "Architect role\n\n- Update docs and cross-service contracts.\n- Enforce compatibility plans for shared interfaces.\n- Emit DocUpdateNeeded when documentation or design is missing."
	case "cell":
		return "Cell role (merged triad)\n\nModes to time-slice within a single agent:\n- Build: implement tasks and update tests.\n- Review: self-review with reviewer-level strictness.\n- Monitor: run checks and emit MonitorRequest as needed.\n\nRules:\n- No out-of-scope edits.\n- Emit events instead of direct cross-cell changes.\n- Keep notes in the outbox with the promise token when done."
	default:
		return ""
	}
}

func ensureHookConfig(worktree string) error {
	path := filepath.Join(worktree, ".mf", "hooks.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	config := `{
  "events": {
    "claude_stop": [],
    "claude_pre_tool": [],
    "claude_permission": [],
    "turn_start": [],
    "turn_end": [],
    "turn_report": []
  }
}`
	return util.AtomicWriteFile(path, []byte(config+"\n"), 0o644)
}
