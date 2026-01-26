package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func Init(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge init <rig> --repo <path>")
	}
	rigName := args[0]
	repo := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--repo" && i+1 < len(args) {
			repo = args[i+1]
			i++
		}
	}
	if strings.TrimSpace(repo) == "" {
		return fmt.Errorf("--repo is required")
	}
	rdir := rig.RigDir(home, rigName)
	if err := util.EnsureDir(rdir); err != nil {
		return err
	}
	cfg := rig.DefaultRigConfig(rigName, repo)
	if err := rig.SaveRigConfig(rig.RigConfigPath(home, rigName), cfg); err != nil {
		return err
	}

	client := beads.Client{RepoPath: repo}
	if err := client.Init(nil); err != nil {
		return err
	}
	if err := ensureBeadsTypes(repo); err != nil {
		return err
	}

	_ = util.EnsureDir(filepath.Join(rdir, "cells"))
	fmt.Printf("Initialized rig %q at %s\n", rigName, rdir)
	fmt.Printf("Beads repo: %s\n", filepath.Join(repo, ".beads"))
	warnDuplicateRepo(home, repo, rigName)
	return nil
}

func warnDuplicateRepo(home, repo, rigName string) {
	rigsDir := filepath.Join(home, "rigs")
	entries, err := os.ReadDir(rigsDir)
	if err != nil {
		return
	}
	repo = filepath.Clean(repo)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == rigName {
			continue
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, name))
		if err != nil {
			continue
		}
		if filepath.Clean(cfg.RepoPath) == repo {
			fmt.Printf("Warning: rig %q already points to %s\n", name, repo)
		}
	}
}
