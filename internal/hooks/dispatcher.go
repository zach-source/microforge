package hooks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type HookAction struct {
	Command      string   `json:"command"`
	OnlyRoles    []string `json:"only_roles,omitempty"`
	OnlyCells    []string `json:"only_cells,omitempty"`
	TimeoutSec   int      `json:"timeout_sec,omitempty"`
	ContinueOnEr bool     `json:"continue_on_error,omitempty"`
}

type HookConfig struct {
	Events map[string][]HookAction `json:"events"`
}

func LoadHookConfig(worktree string) (HookConfig, error) {
	path := filepath.Join(worktree, ".mf", "hooks.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return HookConfig{}, err
	}
	var cfg HookConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return HookConfig{}, err
	}
	if cfg.Events == nil {
		cfg.Events = map[string][]HookAction{}
	}
	return cfg, nil
}

func DispatchHook(event string, payload map[string]any, identity AgentIdentity) error {
	if strings.TrimSpace(identity.Worktree) == "" {
		return nil
	}
	cfg, err := LoadHookConfig(identity.Worktree)
	if err != nil {
		return nil
	}
	actions := cfg.Events[event]
	for _, action := range actions {
		if !matchesSelector(action, identity) {
			continue
		}
		if err := runHook(action, event, payload, identity); err != nil {
			if action.ContinueOnEr {
				continue
			}
			return err
		}
	}
	return nil
}

func matchesSelector(action HookAction, identity AgentIdentity) bool {
	if len(action.OnlyRoles) > 0 {
		ok := false
		for _, role := range action.OnlyRoles {
			if strings.EqualFold(role, identity.Role) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(action.OnlyCells) > 0 {
		ok := false
		for _, cell := range action.OnlyCells {
			if strings.EqualFold(cell, identity.CellName) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func runHook(action HookAction, event string, payload map[string]any, identity AgentIdentity) error {
	if strings.TrimSpace(action.Command) == "" {
		return nil
	}
	timeout := time.Duration(action.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	cmd := exec.Command("sh", "-c", action.Command)
	cmd.Env = append(os.Environ(),
		"MF_EVENT="+event,
		"MF_RIG="+identity.RigName,
		"MF_CELL="+identity.CellName,
		"MF_ROLE="+identity.Role,
		"MF_REPO="+identity.RepoPath,
		"MF_WORKTREE="+identity.Worktree,
	)
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			cmd.Env = append(cmd.Env, "MF_PAYLOAD="+string(b))
		}
	}
	timer := time.AfterFunc(timeout, func() {
		_ = cmd.Process.Kill()
	})
	defer timer.Stop()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
