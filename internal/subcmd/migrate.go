package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func Migrate(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge migrate <beads|rig> [--all]")
	}
	op := args[0]
	rest := args[1:]
	if op != "beads" && op != "rig" {
		return fmt.Errorf("unknown migrate subcommand: %s", op)
	}
	all := false
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--all" {
			all = true
		}
	}
	if all {
		if op == "rig" {
			return migrateAllRigs(home)
		}
		return migrateAllBeads(home)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge migrate %s [--all]", op)
	}
	rigName := rest[0]
	if op == "rig" {
		return migrateRig(home, rigName)
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	if err := ensureBeadsTypes(cfg.RepoPath); err != nil {
		return err
	}
	fmt.Printf("Beads types ensured for %s\n", rigName)
	return nil
}

func migrateAllBeads(home string) error {
	rigDir := filepath.Join(home, "rigs")
	entries, err := os.ReadDir(rigDir)
	if err != nil {
		return err
	}
	updated := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, name))
		if err != nil {
			continue
		}
		if strings.TrimSpace(cfg.RepoPath) == "" {
			continue
		}
		if err := ensureBeadsTypes(cfg.RepoPath); err != nil {
			return err
		}
		updated++
	}
	fmt.Printf("Beads types ensured for %d rig(s)\n", updated)
	return nil
}

func migrateAllRigs(home string) error {
	rigDir := filepath.Join(home, "rigs")
	entries, err := os.ReadDir(rigDir)
	if err != nil {
		return err
	}
	updated := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if err := migrateRig(home, name); err != nil {
			return err
		}
		updated++
	}
	fmt.Printf("Rig settings migrated for %d rig(s)\n", updated)
	return nil
}

func migrateRig(home, rigName string) error {
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	updated := false
	cfg.RuntimeArgs, updated = normalizeRuntimeArgs(cfg.RuntimeProvider, cfg.RuntimeCmd, cfg.RuntimeArgs)
	for role, spec := range cfg.RuntimeRoles {
		args, changed := normalizeRuntimeArgs(cfg.RuntimeProvider, cfg.RuntimeCmd, spec.Args)
		if changed {
			spec.Args = args
			cfg.RuntimeRoles[role] = spec
			updated = true
		}
	}
	if updated {
		if err := rig.SaveRigConfig(rig.RigConfigPath(home, rigName), cfg); err != nil {
			return err
		}
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	for _, cell := range cells {
		wt := cell.WorktreePath
		if err := ensureHookConfig(wt); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(wt, ".claude", "settings.json")); os.IsNotExist(err) {
			_ = util.EnsureDir(filepath.Join(wt, ".claude"))
			settings := `{
  "permissions": { "allow": ["Bash", "Read", "Write", "Edit"] }
}`
			_ = util.AtomicWriteFile(filepath.Join(wt, ".claude", "settings.json"), []byte(settings+"\n"), 0o644)
		}
		for _, p := range []string{"mail/inbox", "mail/outbox", "mail/archive"} {
			_ = util.EnsureDir(filepath.Join(wt, p))
		}
	}
	fmt.Printf("Rig migrated %s\n", rigName)
	return nil
}

func normalizeRuntimeArgs(provider, cmd string, args []string) ([]string, bool) {
	changed := false
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--resume" || args[i] == "--continue" || args[i] == "--fork-session" {
			changed = true
			continue
		}
		if args[i] == "--session-id" {
			changed = true
			if i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, args[i])
	}
	isClaude := strings.EqualFold(provider, "claude") || strings.Contains(strings.ToLower(cmd), "claude")
	if isClaude {
		if !hasArg(out, "--dangerously-skip-permissions") {
			out = append(out, "--dangerously-skip-permissions")
			changed = true
		}
		if hasArg(out, "--resume") || hasArg(out, "--continue") {
			if !hasArg(out, "--fork-session") {
				out = append(out, "--fork-session")
				changed = true
			}
		}
	}
	return out, changed
}
