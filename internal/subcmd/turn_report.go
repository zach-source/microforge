package subcmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
	"github.com/example/microforge/internal/util"
)

func computeTurnSummary(home, rigName string, cfg rig.RigConfig, state turn.State) (turn.Summary, error) {
	return turn.Compute(nil, home, rigName, cfg, state)
}

func printTurnSummary(summary turn.Summary) {
	name := summary.Name
	if strings.TrimSpace(name) == "" {
		name = "-"
	}
	started := summary.StartedAt
	if started == "" {
		started = "-"
	}
	status := "active"
	if summary.EndedAt != "" {
		status = "ended"
	}
	fmt.Printf("Turn %s (%s) %s\n", summary.ID, name, status)
	fmt.Printf("Started: %s (%s ago)\n", started, humanDuration(summary.Duration))
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Commits: %d (+%d / -%d)\n", summary.Commits, summary.Added, summary.Removed)
	fmt.Printf("  Tasks: %d completed, %d open\n", summary.TasksDone, summary.TasksOpen)
	fmt.Printf("  Reviews: %d\n", summary.Reviews)
	if len(summary.Cells) > 0 {
		fmt.Printf("\nBy Cell:\n")
		cells := make([]string, 0, len(summary.Cells))
		for name := range summary.Cells {
			cells = append(cells, name)
		}
		sort.Strings(cells)
		for _, cell := range cells {
			fmt.Printf("  %s\t%d commits\n", cell, summary.Cells[cell])
		}
	}
}

func writeTurnReport(repo string, summary turn.Summary) (string, error) {
	path := turn.ReportPath(repo, summary.ID)
	if err := util.EnsureDir(filepath.Dir(path)); err != nil {
		return "", err
	}
	name := summary.Name
	if strings.TrimSpace(name) == "" {
		name = summary.ID
	}
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "# Turn Report: %s\n\n", name)
	fmt.Fprintf(buf, "**Duration**: %s\n\n", summary.Duration)
	fmt.Fprintf(buf, "**Commits**: %d (+%d / -%d)\n\n", summary.Commits, summary.Added, summary.Removed)
	fmt.Fprintf(buf, "**Tasks**: %d completed, %d open\n\n", summary.TasksDone, summary.TasksOpen)
	fmt.Fprintf(buf, "**Reviews**: %d\n\n", summary.Reviews)
	if len(summary.Cells) > 0 {
		fmt.Fprintf(buf, "## Cells\n")
		cells := make([]string, 0, len(summary.Cells))
		for name := range summary.Cells {
			cells = append(cells, name)
		}
		sort.Strings(cells)
		for _, cell := range cells {
			fmt.Fprintf(buf, "- %s: %d commits\n", cell, summary.Cells[cell])
		}
		fmt.Fprintln(buf)
	}
	if err := util.AtomicWriteFile(path, []byte(buf.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Truncate(time.Second).String()
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}
