package subcmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/store"
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
	if strings.TrimSpace(repo) == "" { return fmt.Errorf("--repo is required") }
	rdir := store.RigDir(home, rigName)
	if err := util.EnsureDir(rdir); err != nil { return err }
	cfg := store.DefaultRigConfig(rigName, repo)
	if err := store.SaveRigConfig(store.RigConfigPath(home, rigName), cfg); err != nil { return err }

	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil { return err }
	defer db.Close()
	if _, err := store.EnsureRig(db, cfg); err != nil { return err }

	_ = util.EnsureDir(filepath.Join(rdir, "cells"))
	fmt.Printf("Initialized rig %q at %s\n", rigName, rdir)
	fmt.Printf("DB: %s\n", store.DBPath(home, rigName))
	return nil
}
