package rig

import (
	"os"
	"path/filepath"
	"strings"
)

func DefaultHome() string {
	home, _ := os.UserHomeDir()
	if strings.TrimSpace(home) == "" {
		return ".microforge"
	}
	return filepath.Join(home, ".microforge")
}

func RigDir(home, rig string) string        { return filepath.Join(home, "rigs", rig) }
func RigConfigPath(home, rig string) string { return filepath.Join(RigDir(home, rig), "rig.json") }
func CellsDir(home, rig string) string      { return filepath.Join(RigDir(home, rig), "cells") }
func CellDir(home, rig, cell string) string { return filepath.Join(CellsDir(home, rig), cell) }
func CellWorktreeDir(home, rig, cell string) string {
	return filepath.Join(CellDir(home, rig, cell), "worktree")
}
func CellMetaDir(home, rig, cell string) string {
	return filepath.Join(CellDir(home, rig, cell), ".mf")
}
func CellRoleMetaPath(home, rig, cell, role string) string {
	return filepath.Join(CellMetaDir(home, rig, cell), "agent-"+role+".json")
}
func CellClaudeSettingsPath(home, rig, cell string) string {
	return filepath.Join(CellWorktreeDir(home, rig, cell), ".claude", "settings.json")
}
func CellConfigPath(home, rig, cell string) string {
	return filepath.Join(CellDir(home, rig, cell), "cell.json")
}
func TurnStatePath(home, rig string) string  { return filepath.Join(RigDir(home, rig), "turn.json") }
func TurnHistoryDir(home, rig string) string { return filepath.Join(RigDir(home, rig), "turns") }
func TurnHistoryPath(home, rig, id string) string {
	return filepath.Join(TurnHistoryDir(home, rig), "turn-"+id+".json")
}
