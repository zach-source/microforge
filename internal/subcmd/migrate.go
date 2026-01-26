package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/rig"
)

func Migrate(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge migrate <beads> [--all]")
	}
	op := args[0]
	rest := args[1:]
	if op != "beads" {
		return fmt.Errorf("unknown migrate subcommand: %s", op)
	}
	all := false
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--all" {
			all = true
		}
	}
	if all {
		return migrateAllBeads(home)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge migrate beads [--all]")
	}
	rigName := rest[0]
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
