package subcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/example/microforge/internal/context"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func Rig(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge rig <list|delete|rename|backup|restore> ...")
	}
	op := args[0]
	rest := args[1:]
	switch op {
	case "list":
		return rigList(home)
	case "delete":
		return rigDelete(home, rest)
	case "rename":
		return rigRename(home, rest)
	case "backup":
		return rigBackup(home, rest)
	case "restore":
		return rigRestore(home, rest)
	default:
		return fmt.Errorf("unknown rig subcommand: %s", op)
	}
}

func rigList(home string) error {
	rigsDir := filepath.Join(home, "rigs")
	entries, err := os.ReadDir(rigsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No rigs")
			return nil
		}
		return err
	}
	state, _ := context.Load(home)
	active := strings.TrimSpace(state.ActiveRig)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		marker := " "
		if name == active {
			marker = "*"
		}
		fmt.Printf("%s %s\n", marker, name)
	}
	return nil
}

func rigDelete(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge rig delete <rig>")
	}
	rigName := args[0]
	rdir := rig.RigDir(home, rigName)
	if err := os.RemoveAll(rdir); err != nil {
		return err
	}
	if s, err := context.Load(home); err == nil {
		if strings.TrimSpace(s.ActiveRig) == rigName {
			_ = context.Clear(home)
		}
	}
	fmt.Printf("Deleted rig %s\n", rigName)
	return nil
}

func rigRename(home string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mforge rig rename <old> <new>")
	}
	oldName := args[0]
	newName := args[1]
	if strings.TrimSpace(newName) == "" {
		return fmt.Errorf("new rig name is required")
	}
	oldDir := rig.RigDir(home, oldName)
	newDir := rig.RigDir(home, newName)
	if _, err := os.Stat(oldDir); err != nil {
		return err
	}
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("rig already exists: %s", newName)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return err
	}
	cfgPath := rig.RigConfigPath(home, newName)
	cfg, err := rig.LoadRigConfig(cfgPath)
	if err == nil {
		cfg.Name = newName
		_ = rig.SaveRigConfig(cfgPath, cfg)
	}
	cells, _ := rig.ListCellConfigs(home, newName)
	for _, cell := range cells {
		updated := false
		if strings.HasPrefix(cell.WorktreePath, oldDir) {
			cell.WorktreePath = strings.Replace(cell.WorktreePath, oldDir, newDir, 1)
			updated = true
		}
		if updated {
			_ = rig.SaveCellConfig(rig.CellConfigPath(home, newName, cell.Name), cell)
		}
		updateRoleMetadata(newName, cell.WorktreePath, oldName)
	}
	if s, err := context.Load(home); err == nil {
		if strings.TrimSpace(s.ActiveRig) == oldName {
			s.ActiveRig = newName
			_ = context.Save(home, s)
		}
	}
	fmt.Printf("Renamed rig %s -> %s\n", oldName, newName)
	return nil
}

func rigBackup(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge rig backup <rig> [--out <path>]")
	}
	rigName := args[0]
	out := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--out" && i+1 < len(args) {
			out = args[i+1]
			i++
		}
	}
	rdir := rig.RigDir(home, rigName)
	if out == "" {
		backupDir := filepath.Join(home, "backups")
		_ = util.EnsureDir(backupDir)
		ts := time.Now().UTC().Format("20060102T150405Z")
		out = filepath.Join(backupDir, fmt.Sprintf("rig-%s-%s.tar.gz", rigName, ts))
	}
	if err := createTarGz(out, rdir); err != nil {
		return err
	}
	fmt.Printf("Backup written to %s\n", out)
	return nil
}

func rigRestore(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge rig restore <archive> [--name <rig>] [--force]")
	}
	archive := args[0]
	name := ""
	force := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--force":
			force = true
		}
	}
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	dest := rig.RigDir(home, name)
	if _, err := os.Stat(dest); err == nil {
		if !force {
			return fmt.Errorf("rig already exists: %s (use --force to overwrite)", name)
		}
		_ = os.RemoveAll(dest)
	}
	if err := extractTarGz(archive, dest); err != nil {
		return err
	}
	cells, _ := rig.ListCellConfigs(home, name)
	for _, cell := range cells {
		updateRoleMetadata(name, cell.WorktreePath, "")
	}
	cfgPath := rig.RigConfigPath(home, name)
	cfg, err := rig.LoadRigConfig(cfgPath)
	if err == nil && cfg.Name != name {
		cfg.Name = name
		_ = rig.SaveRigConfig(cfgPath, cfg)
	}
	fmt.Printf("Restored rig %s from %s\n", name, archive)
	return nil
}

func createTarGz(outPath, srcDir string) error {
	if err := util.EnsureDir(filepath.Dir(outPath)); err != nil {
		return err
	}
	cmd := exec.Command("tar", "-czf", outPath, "-C", filepath.Dir(srcDir), filepath.Base(srcDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func extractTarGz(archive, destDir string) error {
	if err := util.EnsureDir(filepath.Dir(destDir)); err != nil {
		return err
	}
	base := filepath.Base(destDir)
	parent := filepath.Dir(destDir)
	cmd := exec.Command("tar", "-xzf", archive, "-C", parent)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	// If archive name doesn't match desired dest, rename.
	if _, err := os.Stat(destDir); err == nil {
		return nil
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == base {
			return nil
		}
		candidate := filepath.Join(parent, name)
		if _, err := os.Stat(filepath.Join(candidate, "rig.json")); err == nil {
			return os.Rename(candidate, destDir)
		}
	}
	return fmt.Errorf("restored archive but rig dir not found at %s", destDir)
}

func updateRoleMetadata(rigName, worktree, oldName string) {
	paths := []string{
		filepath.Join(worktree, ".mf", "active-agent.json"),
	}
	roleFiles, _ := filepath.Glob(filepath.Join(worktree, ".mf", "active-agent-*.json"))
	paths = append(paths, roleFiles...)
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		if oldName == "" {
			m["rig_name"] = rigName
		} else if name, ok := m["rig_name"].(string); ok && name == oldName {
			m["rig_name"] = rigName
		}
		if home, ok := m["rig_home"].(string); ok && home != "" {
			// keep as-is
			_ = home
		}
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			continue
		}
		_ = util.AtomicWriteFile(p, out, 0o644)
	}
}
