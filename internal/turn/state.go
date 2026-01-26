package turn

import (
	"encoding/json"
	"os"
)

type State struct {
	ID        string `json:"id"`
	StartedAt string `json:"started_at"`
	Status    string `json:"status"`
}

func Load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

func Save(path string, s State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
