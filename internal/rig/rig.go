// Package rig provides configuration types and utilities for Microforge rigs and cells.
// A rig is a workspace pointing to a monorepo, and a cell is a unit for one microservice
// scope with its own git worktree.
package rig

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// RigConfig represents the configuration for a Microforge rig, stored in rig.json.
// It defines the monorepo path, tmux naming, runtime provider, and remote execution settings.
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

// RuntimeSpec defines the command and arguments for a specific role's runtime.
type RuntimeSpec struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

// CellConfig represents the configuration for a cell within a rig, stored in cell.json.
// It defines the cell name, scope prefix for path restrictions, and worktree location.
type CellConfig struct {
	Name         string `json:"name"`
	ScopePrefix  string `json:"scope_prefix"`
	WorktreePath string `json:"worktree_path"`
	CreatedAt    string `json:"created_at"`
}

// DefaultRigConfig returns a RigConfig with sensible defaults for local Claude execution.
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

// SaveRigConfig writes the rig configuration to the specified path as JSON.
func SaveRigConfig(path string, cfg RigConfig) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling rig config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("writing rig config %s: %w", path, err)
	}
	return nil
}

// LoadRigConfig reads and parses the rig configuration from the specified path.
// It applies defaults for missing optional fields.
func LoadRigConfig(path string) (RigConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return RigConfig{}, fmt.Errorf("reading rig config %s: %w", path, err)
	}
	var cfg RigConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return RigConfig{}, fmt.Errorf("parsing rig config %s: %w", path, err)
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

// SaveCellConfig writes the cell configuration to the specified path as JSON.
func SaveCellConfig(path string, cfg CellConfig) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cell config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("writing cell config %s: %w", path, err)
	}
	return nil
}

// LoadCellConfig reads and parses the cell configuration from the specified path.
// Returns an error if name or scope_prefix is missing.
func LoadCellConfig(path string) (CellConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return CellConfig{}, fmt.Errorf("reading cell config %s: %w", path, err)
	}
	var cfg CellConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return CellConfig{}, fmt.Errorf("parsing cell config %s: %w", path, err)
	}
	if cfg.Name == "" || cfg.ScopePrefix == "" {
		return CellConfig{}, fmt.Errorf("invalid cell.json: missing name or scope")
	}
	return cfg, nil
}
