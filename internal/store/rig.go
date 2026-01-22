package store

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type RigConfig struct {
	Name      string   `json:"name"`
	RepoPath  string   `json:"repo_path"`
	TmuxPrefix string  `json:"tmux_prefix"`
	RuntimeCmd string  `json:"runtime_cmd"`
	RuntimeArgs []string `json:"runtime_args"`
	CreatedAt string   `json:"created_at"`
}

func DefaultRigConfig(name, repo string) RigConfig {
	return RigConfig{
		Name: name,
		RepoPath: repo,
		TmuxPrefix: "mf",
		RuntimeCmd: "claude",
		RuntimeArgs: []string{"--resume"},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func SaveRigConfig(path string, cfg RigConfig) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil { return err }
	return os.WriteFile(path, b, 0o644)
}

func LoadRigConfig(path string) (RigConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil { return RigConfig{}, err }
	var cfg RigConfig
	if err := json.Unmarshal(b, &cfg); err != nil { return RigConfig{}, err }
	if cfg.Name == "" { return RigConfig{}, fmt.Errorf("invalid rig.json: missing name") }
	if cfg.TmuxPrefix == "" { cfg.TmuxPrefix = "mf" }
	if cfg.RuntimeCmd == "" { cfg.RuntimeCmd = "claude" }
	if len(cfg.RuntimeArgs) == 0 { cfg.RuntimeArgs = []string{"--resume"} }
	return cfg, nil
}
