package rig

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type RigConfig struct {
	Name                 string                 `json:"name"`
	RepoPath             string                 `json:"repo_path"`
	TmuxPrefix           string                 `json:"tmux_prefix"`
	RuntimeProvider      string                 `json:"runtime_provider"`
	RuntimeCmd           string                 `json:"runtime_cmd"`
	RuntimeArgs          []string               `json:"runtime_args"`
	RuntimeRoles         map[string]RuntimeSpec `json:"runtime_roles"`
	RemoteHost           string                 `json:"remote_host"`
	RemoteUser           string                 `json:"remote_user"`
	RemotePort           int                    `json:"remote_port"`
	RemoteWorkdir        string                 `json:"remote_workdir"`
	RemoteTmuxPrefix     string                 `json:"remote_tmux_prefix"`
	LibraryAddr          string                 `json:"library_addr"`
	LibraryDocs          []string               `json:"library_docs"`
	LibraryContext7URL   string                 `json:"library_context7_url"`
	LibraryContext7Token string                 `json:"library_context7_token"`
	CreatedAt            string                 `json:"created_at"`
}

type RuntimeSpec struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

type CellConfig struct {
	Name         string `json:"name"`
	ScopePrefix  string `json:"scope_prefix"`
	WorktreePath string `json:"worktree_path"`
	CreatedAt    string `json:"created_at"`
}

func DefaultRigConfig(name, repo string) RigConfig {
	return RigConfig{
		Name:            name,
		RepoPath:        repo,
		TmuxPrefix:      "mforge",
		RuntimeProvider: "claude",
		RuntimeCmd:      "claude",
		RuntimeArgs:     []string{"--dangerously-skip-permissions"},
		RuntimeRoles:    map[string]RuntimeSpec{},
		RemotePort:      22,
		LibraryAddr:     "127.0.0.1:7331",
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
}

func SaveRigConfig(path string, cfg RigConfig) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func LoadRigConfig(path string) (RigConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return RigConfig{}, err
	}
	var cfg RigConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return RigConfig{}, err
	}
	if cfg.Name == "" {
		return RigConfig{}, fmt.Errorf("invalid rig.json: missing name")
	}
	if cfg.TmuxPrefix == "" {
		cfg.TmuxPrefix = "mforge"
	}
	if cfg.RuntimeProvider == "" {
		cfg.RuntimeProvider = "claude"
	}
	if cfg.RuntimeCmd == "" {
		cfg.RuntimeCmd = "claude"
	}
	if len(cfg.RuntimeArgs) == 0 {
		cfg.RuntimeArgs = []string{"--dangerously-skip-permissions"}
	}
	if cfg.RuntimeRoles == nil {
		cfg.RuntimeRoles = map[string]RuntimeSpec{}
	}
	if cfg.RemotePort == 0 {
		cfg.RemotePort = 22
	}
	if cfg.LibraryAddr == "" {
		cfg.LibraryAddr = "127.0.0.1:7331"
	}
	return cfg, nil
}

func SaveCellConfig(path string, cfg CellConfig) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func LoadCellConfig(path string) (CellConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return CellConfig{}, err
	}
	var cfg CellConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return CellConfig{}, err
	}
	if cfg.Name == "" || cfg.ScopePrefix == "" {
		return CellConfig{}, fmt.Errorf("invalid cell.json: missing name or scope")
	}
	return cfg, nil
}
