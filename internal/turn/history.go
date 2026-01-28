package turn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/example/microforge/internal/util"
)

type Record struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
}

func SaveRecord(path string, rec Record) error {
	if err := util.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return util.AtomicWriteFile(path, b, 0o644)
}

func LoadRecord(path string) (Record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(b, &rec); err != nil {
		return Record{}, err
	}
	return rec, nil
}

func ListRecords(dir string) ([]Record, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, err
	}
	out := make([]Record, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		rec, err := LoadRecord(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return parseTime(out[i].StartedAt).Before(parseTime(out[j].StartedAt))
	})
	return out, nil
}

func parseTime(val string) time.Time {
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Time{}
	}
	return t
}
