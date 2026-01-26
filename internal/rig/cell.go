package rig

import (
	"os"
	"path/filepath"
)

func ListCellConfigs(home, rigName string) ([]CellConfig, error) {
	cellsDir := CellsDir(home, rigName)
	entries, err := os.ReadDir(cellsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []CellConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cellName := entry.Name()
		cfg, err := LoadCellConfig(filepath.Join(cellsDir, cellName, "cell.json"))
		if err != nil {
			continue
		}
		out = append(out, cfg)
	}
	return out, nil
}
