package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func Checkpoint(home string, args []string) error {
	message := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--message" && i+1 < len(args) {
			message = args[i+1]
			i++
		}
	}
	if strings.TrimSpace(message) == "" {
		message = fmt.Sprintf("checkpoint %s", time.Now().UTC().Format(time.RFC3339))
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge checkpoint [--message <text>]")
	}
	rigName := args[0]
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(cfg.RepoPath, ".git")); err != nil {
		return fmt.Errorf("no git repo found at %s", cfg.RepoPath)
	}
	status, err := util.RunInDir(nil, cfg.RepoPath, "git", "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status.Stdout) == "" {
		fmt.Println("No changes to checkpoint")
		return nil
	}
	if _, err := util.RunInDir(nil, cfg.RepoPath, "git", "add", "-A"); err != nil {
		return err
	}
	if _, err := util.RunInDir(nil, cfg.RepoPath, "git", "commit", "-m", message); err != nil {
		return err
	}
	fmt.Printf("Checkpointed repo: %s\n", message)
	return nil
}
