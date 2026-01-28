package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
	"github.com/example/microforge/internal/util"
)

func Turn(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge turn <start|status|end|slate|list|diff> <rig>")
	}
	op := args[0]
	rest := args[1:]
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge turn %s <rig>", op)
	}
	rigName := rest[0]
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	statePath := rig.TurnStatePath(home, rigName)
	switch op {
	case "start":
		name := ""
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--name" && i+1 < len(rest) {
				name = rest[i+1]
				i++
			}
		}
		if err := closeActiveTurn(home, rigName, cfg, client, false); err != nil {
			return err
		}
		title := fmt.Sprintf("Turn %s", time.Now().UTC().Format(time.RFC3339))
		issue, err := client.Create(nil, beads.CreateRequest{
			Title:       title,
			Type:        "turn",
			Priority:    "p1",
			Status:      "open",
			Description: "turn-start",
		})
		if err != nil {
			return err
		}
		state := turn.State{ID: issue.ID, StartedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Name: name}
		if err := util.EnsureDir(filepath.Dir(statePath)); err != nil {
			return err
		}
		if err := turn.Save(statePath, state); err != nil {
			return err
		}
		fmt.Printf("Started turn %s (%s)\n", issue.ID, title)
		return nil
	case "status":
		state, err := turn.Load(statePath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No active turn")
				return nil
			}
			return err
		}
		summary, err := computeTurnSummary(home, rigName, cfg, state)
		if err != nil {
			return err
		}
		printTurnSummary(summary)
		return nil
	case "end":
		report := false
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--report" {
				report = true
			}
		}
		return closeActiveTurn(home, rigName, cfg, client, report)
	case "slate":
		state, err := turn.Load(statePath)
		if err != nil {
			return err
		}
		issues, err := client.List(nil)
		if err != nil {
			return err
		}
		fmt.Printf("Turn slate %s\n", state.ID)
		for _, issue := range issues {
			meta := beads.ParseMeta(issue.Description)
			if meta.TurnID != state.ID {
				continue
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", issue.ID, issue.Status, issue.Type, strings.ReplaceAll(issue.Title, "\t", " "))
		}
		return nil
	case "list":
		return listTurns(home, rigName)
	case "diff":
		id := ""
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--id" && i+1 < len(rest) {
				id = rest[i+1]
				i++
			}
		}
		return diffTurn(home, rigName, cfg, id)
	default:
		return fmt.Errorf("unknown turn subcommand: %s", op)
	}
}

func closeActiveTurn(home, rigName string, cfg rig.RigConfig, client beads.Client, report bool) error {
	statePath := rig.TurnStatePath(home, rigName)
	state, err := turn.Load(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No active turn")
			return nil
		}
		return err
	}
	if strings.TrimSpace(state.ID) != "" {
		_, _ = client.Close(nil, state.ID, "turn complete")
	}
	ended := time.Now().UTC().Format(time.RFC3339)
	rec := turn.Record{ID: state.ID, Name: state.Name, StartedAt: state.StartedAt, EndedAt: ended}
	if err := turn.SaveRecord(rig.TurnHistoryPath(home, rigName, state.ID), rec); err != nil {
		return err
	}
	if report {
		summary, err := computeTurnSummary(home, rigName, cfg, state)
		if err != nil {
			return err
		}
		path, err := writeTurnReport(cfg.RepoPath, summary)
		if err != nil {
			return err
		}
		fmt.Printf("Report saved to %s\n", path)
	}
	_ = os.Remove(statePath)
	fmt.Printf("Ended turn %s\n", state.ID)
	return nil
}

func listTurns(home, rigName string) error {
	recs, err := turn.ListRecords(rig.TurnHistoryDir(home, rigName))
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		fmt.Println("No turns found")
		return nil
	}
	for _, rec := range recs {
		duration := "-"
		start := parseRFC3339(rec.StartedAt)
		end := parseRFC3339(rec.EndedAt)
		if !start.IsZero() && !end.IsZero() {
			duration = end.Sub(start).String()
		}
		name := rec.Name
		if name == "" {
			name = "-"
		}
		fmt.Printf("%s\t%s\t%s\n", rec.ID, name, duration)
	}
	return nil
}

func diffTurn(home, rigName string, cfg rig.RigConfig, id string) error {
	rec, err := findTurnRecord(home, rigName, id)
	if err != nil {
		return err
	}
	start := rec.StartedAt
	end := rec.EndedAt
	if end == "" {
		end = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := util.Run(nil, "git", "-C", cfg.RepoPath, "log", "--since", start, "--until", end, "--stat")
	if err != nil {
		return err
	}
	fmt.Println(res.Stdout)
	return nil
}

func findTurnRecord(home, rigName, id string) (turn.Record, error) {
	if strings.TrimSpace(id) != "" {
		path := rig.TurnHistoryPath(home, rigName, id)
		if rec, err := turn.LoadRecord(path); err == nil {
			return rec, nil
		}
	}
	statePath := rig.TurnStatePath(home, rigName)
	state, err := turn.Load(statePath)
	if err == nil {
		return turn.Record{ID: state.ID, Name: state.Name, StartedAt: state.StartedAt, EndedAt: state.EndedAt}, nil
	}
	return turn.Record{}, fmt.Errorf("turn not found")
}

func parseRFC3339(val string) time.Time {
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Time{}
	}
	return t
}
