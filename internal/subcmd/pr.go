package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func PR(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge pr <create|ready|link-review> ...")
	}
	op := args[0]
	rest := args[1:]
	switch op {
	case "create":
		return prCreate(home, rest)
	case "ready":
		return prReady(home, rest)
	case "link-review":
		return prLinkReview(home, rest)
	default:
		return fmt.Errorf("unknown pr subcommand: %s", op)
	}
}

func prCreate(home string, rest []string) error {
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge pr create <rig> --title <title> --cell <cell> [--url <url>] [--turn <id>] [--status <status>]")
	}
	rigName := rest[0]
	var title, cell, url, turnID, status string
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--title":
			if i+1 < len(rest) {
				title = rest[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(rest) {
				cell = rest[i+1]
				i++
			}
		case "--url":
			if i+1 < len(rest) {
				url = rest[i+1]
				i++
			}
		case "--turn":
			if i+1 < len(rest) {
				turnID = rest[i+1]
				i++
			}
		case "--status":
			if i+1 < len(rest) {
				status = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(title) == "" || strings.TrimSpace(cell) == "" {
		return fmt.Errorf("--title and --cell are required")
	}
	if strings.TrimSpace(turnID) == "" {
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
	}
	if err := beadLimit(home, rigName, cell, turnID); err != nil {
		return err
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	meta := beads.Meta{Cell: cell, TurnID: turnID}
	body := ""
	if strings.TrimSpace(url) != "" {
		body = "PR: " + url
	}
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "pr",
		Priority:    "p1",
		Status:      defaultIfEmpty(status, "open"),
		Description: beads.RenderMeta(meta) + "\n\n" + body,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Created PR bead %s\n", issue.ID)
	return nil
}

func prReady(home string, rest []string) error {
	if len(rest) < 2 {
		return fmt.Errorf("usage: mforge pr ready <rig> <id>")
	}
	rigName, id := rest[0], rest[1]
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	_, err = client.UpdateStatus(nil, id, "ready")
	return err
}

func prLinkReview(home string, rest []string) error {
	if len(rest) < 3 {
		return fmt.Errorf("usage: mforge pr link-review <rig> <pr_id> <review_id>")
	}
	rigName, prID, reviewID := rest[0], rest[1], rest[2]
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	return client.DepAdd(nil, prID, "review:"+reviewID)
}
