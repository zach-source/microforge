package context

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/example/microforge/internal/util"
)

type State struct {
	ActiveRig string `json:"active_rig"`
}

func Path(home string) string {
	return filepath.Join(home, "context.json")
}

func Load(home string) (State, error) {
	b, err := os.ReadFile(Path(home))
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

func Save(home string, s State) error {
	path := Path(home)
	if err := util.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return util.AtomicWriteFile(path, b, 0o644)
}

func Clear(home string) error {
	return os.Remove(Path(home))
}
