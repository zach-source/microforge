package turn

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

type Summary struct {
	ID        string
	Name      string
	StartedAt string
	EndedAt   string
	Duration  time.Duration
	Commits   int
	Added     int
	Removed   int
	TasksDone int
	TasksOpen int
	Reviews   int
	Cells     map[string]int
	Recent    []string
}

func Compute(ctx context.Context, home, rigName string, cfg rig.RigConfig, state State) (Summary, error) {
	start := parseRFC3339(state.StartedAt)
	end := time.Now().UTC()
	if state.EndedAt != "" {
		if t := parseRFC3339(state.EndedAt); !t.IsZero() {
			end = t
		}
	}
	summary := Summary{
		ID:        state.ID,
		Name:      state.Name,
		StartedAt: state.StartedAt,
		EndedAt:   state.EndedAt,
		Duration:  end.Sub(start),
		Cells:     map[string]int{},
	}
	if err := fillGitSummary(ctx, cfg.RepoPath, &summary, start, end); err != nil {
		return summary, err
	}
	if err := fillBeadsSummary(ctx, cfg.RepoPath, &summary, start, end); err != nil {
		return summary, err
	}
	if err := fillCellSummary(ctx, home, rigName, cfg.RepoPath, &summary, start, end); err != nil {
		return summary, err
	}
	return summary, nil
}

func fillGitSummary(ctx context.Context, repo string, summary *Summary, start, end time.Time) error {
	res, err := util.RunInDir(ctx, repo, "git", "log", "--since", start.Format(time.RFC3339), "--until", end.Format(time.RFC3339), "--numstat", "--pretty=%H")
	if err != nil {
		return nil
	}
	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 1 {
			summary.Commits++
			continue
		}
		if len(parts) >= 2 {
			added := parseInt(parts[0])
			removed := parseInt(parts[1])
			summary.Added += added
			summary.Removed += removed
		}
	}
	return nil
}

func fillBeadsSummary(ctx context.Context, repo string, summary *Summary, start, end time.Time) error {
	client := beads.Client{RepoPath: repo}
	issues, err := client.List(ctx)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		created := parseRFC3339(issue.CreatedAt)
		if created.IsZero() {
			continue
		}
		if created.Before(start) || created.After(end) {
			continue
		}
		switch strings.ToLower(issue.Type) {
		case "task":
			if strings.ToLower(issue.Status) == "done" || strings.ToLower(issue.Status) == "closed" {
				summary.TasksDone++
			} else {
				summary.TasksOpen++
			}
		case "review":
			summary.Reviews++
		}
	}
	return nil
}

func fillCellSummary(ctx context.Context, home, rigName, repo string, summary *Summary, start, end time.Time) error {
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	for _, cell := range cells {
		if _, err := util.RunInDir(ctx, cell.WorktreePath, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
			continue
		}
		res, err := util.RunInDir(ctx, cell.WorktreePath, "git", "log", "--since", start.Format(time.RFC3339), "--until", end.Format(time.RFC3339), "--pretty=oneline")
		if err != nil {
			continue
		}
		count := 0
		for _, line := range strings.Split(res.Stdout, "\n") {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		if count > 0 {
			summary.Cells[cell.Name] = count
		}
	}
	return nil
}

func parseRFC3339(val string) time.Time {
	if strings.TrimSpace(val) == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(val))
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseInt(val string) int {
	if val == "-" {
		return 0
	}
	var out int
	_, _ = fmt.Sscanf(val, "%d", &out)
	return out
}

func ReportPath(repo, id string) string {
	return filepath.Join(repo, ".mforge", "reports", "turn-"+id+".md")
}
