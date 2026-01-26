package subcmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

type requestPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Kind  string `json:"kind"`
	Scope string `json:"scope"`
}

func Request(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge request <create|list|triage> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "create":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge request create <rig> --cell <cell> --role <role> --severity <sev> --priority <p> --scope <path> --payload <json>")
		}
		rigName := rest[0]
		var cellName, role, severity, priority, scope, payload string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--cell":
				if i+1 < len(rest) {
					cellName = rest[i+1]
					i++
				}
			case "--role":
				if i+1 < len(rest) {
					role = rest[i+1]
					i++
				}
			case "--severity":
				if i+1 < len(rest) {
					severity = rest[i+1]
					i++
				}
			case "--priority":
				if i+1 < len(rest) {
					priority = rest[i+1]
					i++
				}
			case "--scope":
				if i+1 < len(rest) {
					scope = rest[i+1]
					i++
				}
			case "--payload":
				if i+1 < len(rest) {
					payload = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(cellName) == "" || strings.TrimSpace(role) == "" {
			return fmt.Errorf("--cell and --role are required")
		}
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		turnID := ""
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
		if err := beadLimit(home, rigName, cellName, turnID); err != nil {
			return err
		}
		meta := beads.Meta{
			Cell:       cellName,
			SourceRole: role,
			Scope:      scope,
			Kind:       "request",
			Severity:   severity,
			TurnID:     turnID,
		}
		desc := beads.RenderMeta(meta) + "\n\n" + payload
		title := "Request from " + role
		payloadData := requestPayload{}
		if err := json.Unmarshal([]byte(payload), &payloadData); err == nil {
			if strings.TrimSpace(payloadData.Title) != "" {
				title = payloadData.Title
			}
		}
		req := beads.CreateRequest{
			Title:       title,
			Type:        "request",
			Priority:    priority,
			Status:      "open",
			Description: desc,
		}
		issue, err := client.Create(nil, req)
		if err != nil {
			return err
		}
		fmt.Printf("Created request %s\n", issue.ID)
		return nil

	case "list":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge request list <rig> [--cell <cell>] [--status <status>] [--priority <p>]")
		}
		rigName := rest[0]
		var cellName, status, priority string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--cell":
				if i+1 < len(rest) {
					cellName = rest[i+1]
					i++
				}
			case "--status":
				if i+1 < len(rest) {
					status = rest[i+1]
					i++
				}
			case "--priority":
				if i+1 < len(rest) {
					priority = rest[i+1]
					i++
				}
			}
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		list, err := client.List(nil)
		if err != nil {
			return err
		}
		for _, r := range list {
			if strings.ToLower(r.Type) != "request" {
				continue
			}
			if strings.TrimSpace(status) != "" && r.Status != status {
				continue
			}
			if strings.TrimSpace(priority) != "" && r.Priority != priority {
				continue
			}
			meta := beads.ParseMeta(r.Description)
			if strings.TrimSpace(cellName) != "" && meta.Cell != cellName {
				continue
			}
			scope := meta.Scope
			fmt.Printf("%s\t%s\t%s\t%s\t%s", r.ID, r.Status, r.Priority, meta.SourceRole, meta.Cell)
			if scope != "" {
				fmt.Printf("\t(scope=%s)", scope)
			}
			fmt.Println()
		}
		return nil

	case "triage":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge request triage <rig> --request <id> --action create-task|merge|block")
		}
		rigName := rest[0]
		var reqID, action string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--request":
				if i+1 < len(rest) {
					reqID = rest[i+1]
					i++
				}
			case "--action":
				if i+1 < len(rest) {
					action = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(reqID) == "" || strings.TrimSpace(action) == "" {
			return fmt.Errorf("--request and --action are required")
		}
		cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: cfg.RepoPath}
		reqIssue, err := client.Show(nil, reqID)
		if err != nil {
			return err
		}

		switch action {
		case "create-task":
			meta := beads.ParseMeta(reqIssue.Description)
			payload := requestPayload{}
			_ = json.Unmarshal([]byte(beads.StripMeta(reqIssue.Description)), &payload)
			title := strings.TrimSpace(payload.Title)
			if title == "" {
				title = fmt.Sprintf("Request %s", reqIssue.ID)
			}
			body := strings.TrimSpace(payload.Body)
			if body == "" {
				body = beads.StripMeta(reqIssue.Description)
			}
			kind := strings.TrimSpace(payload.Kind)
			scope := payload.Scope
			if scope == "" {
				scope = meta.Scope
			}
			taskMeta := beads.Meta{Cell: meta.Cell, Scope: scope, Kind: kind, Title: title}
			taskDesc := beads.RenderMeta(taskMeta) + "\n\n" + body
			taskIssue, err := client.Create(nil, beads.CreateRequest{
				Title:       title,
				Type:        "task",
				Priority:    reqIssue.Priority,
				Status:      "open",
				Description: taskDesc,
				Deps:        []string{"related:" + reqIssue.ID},
			})
			if err != nil {
				return err
			}
			cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, meta.Cell))
			if err != nil {
				return err
			}
			inboxRel := fmt.Sprintf("mail/inbox/%s.md", taskIssue.ID)
			outboxRel := fmt.Sprintf("mail/outbox/%s.md", taskIssue.ID)
			turnID := ""
			if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
				turnID = strings.TrimSpace(state.ID)
			}
			if err := beadLimit(home, rigName, meta.Cell, turnID); err != nil {
				return err
			}
			assnMeta := beads.Meta{
				Cell:     meta.Cell,
				Role:     "builder",
				Scope:    scope,
				Inbox:    inboxRel,
				Outbox:   outboxRel,
				Promise:  "DONE",
				TurnID:   turnID,
				Worktree: cellCfg.WorktreePath,
			}
			assnDesc := beads.RenderMeta(assnMeta) + "\n\n" + title
			if _, err := client.Create(nil, beads.CreateRequest{
				Title:       "Assignment " + taskIssue.ID,
				Type:        "assignment",
				Priority:    reqIssue.Priority,
				Status:      "open",
				Description: assnDesc,
				Deps:        []string{"related:" + taskIssue.ID},
			}); err != nil {
				return err
			}
			_, _ = client.UpdateStatus(nil, reqIssue.ID, "in_progress")
			fmt.Printf("Triaged request %s -> task %s\n", reqIssue.ID, taskIssue.ID)
			return nil
		case "merge":
			_, err := client.Close(nil, reqIssue.ID, "merged")
			return err
		case "block":
			_, err := client.UpdateStatus(nil, reqIssue.ID, "blocked")
			return err
		default:
			return fmt.Errorf("unknown action: %s", action)
		}

	default:
		return fmt.Errorf("unknown request subcommand: %s", op)
	}
}
