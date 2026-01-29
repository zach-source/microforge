package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func Bead(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge bead <create|list|show|close|triage|dep|status> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "create":
		return beadCreate(home, rest)
	case "list":
		return beadList(home, rest)
	case "show":
		return beadShow(home, rest)
	case "close":
		return beadClose(home, rest)
	case "triage":
		return beadTriage(home, rest)
	case "dep":
		return beadDep(home, rest)
	case "status":
		return beadStatus(home, rest)
	default:
		return fmt.Errorf("unknown bead subcommand: %s", op)
	}
}

func beadCreate(home string, rest []string) error {
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge bead create --type <type> --title <title> [--priority <p>] [--status <status>] [--cell <cell>] [--role <role>] [--scope <path>] [--turn <id>] [--severity <sev>] [--description <text>] [--acceptance <text>] [--compat <text>] [--links <text>] [--deps <a,b,c>]")
	}
	rigName := rest[0]
	var beadType, title, priority, status, cell, role, scope, turnID, severity, desc, depsCSV, acceptance, compat, links string
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--type":
			if i+1 < len(rest) {
				beadType = rest[i+1]
				i++
			}
		case "--title":
			if i+1 < len(rest) {
				title = rest[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(rest) {
				priority = rest[i+1]
				i++
			}
		case "--status":
			if i+1 < len(rest) {
				status = rest[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(rest) {
				cell = rest[i+1]
				i++
			}
		case "--role":
			if i+1 < len(rest) {
				role = rest[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(rest) {
				scope = rest[i+1]
				i++
			}
		case "--turn":
			if i+1 < len(rest) {
				turnID = rest[i+1]
				i++
			}
		case "--severity":
			if i+1 < len(rest) {
				severity = rest[i+1]
				i++
			}
		case "--description":
			if i+1 < len(rest) {
				desc = rest[i+1]
				i++
			}
		case "--acceptance":
			if i+1 < len(rest) {
				acceptance = rest[i+1]
				i++
			}
		case "--compat":
			if i+1 < len(rest) {
				compat = rest[i+1]
				i++
			}
		case "--links":
			if i+1 < len(rest) {
				links = rest[i+1]
				i++
			}
		case "--deps":
			if i+1 < len(rest) {
				depsCSV = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(beadType) == "" || strings.TrimSpace(title) == "" {
		return fmt.Errorf("--type and --title are required")
	}
	if strings.TrimSpace(turnID) == "" {
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
	}
	if err := beadLimit(home, rigName, cell, turnID); err != nil {
		return fmt.Errorf("checking bead limit: %w", err)
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return fmt.Errorf("loading rig %s: %w", rigName, err)
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	meta := beads.Meta{
		Cell:     cell,
		Role:     role,
		Scope:    scope,
		TurnID:   turnID,
		Severity: severity,
	}
	fullDesc := beads.RenderMeta(meta)
	body := renderTemplate(beadType, desc, acceptance, compat, links)
	if strings.TrimSpace(body) != "" {
		fullDesc += "\n\n" + body
	}
	deps := splitCSV(depsCSV)
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        beadType,
		Priority:    priority,
		Status:      status,
		Description: fullDesc,
		Deps:        deps,
	})
	if err != nil {
		return fmt.Errorf("creating bead: %w", err)
	}
	fmt.Printf("Created bead %s\n", issue.ID)
	return nil
}

func beadList(home string, rest []string) error {
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge bead list [--type <type>] [--status <status>] [--cell <cell>] [--priority <p>] [--turn <id>]")
	}
	rigName := rest[0]
	var beadType, status, cell, priority, turnID string
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--type":
			if i+1 < len(rest) {
				beadType = rest[i+1]
				i++
			}
		case "--status":
			if i+1 < len(rest) {
				status = rest[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(rest) {
				cell = rest[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(rest) {
				priority = rest[i+1]
				i++
			}
		case "--turn":
			if i+1 < len(rest) {
				turnID = rest[i+1]
				i++
			}
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return fmt.Errorf("loading rig %s: %w", rigName, err)
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return fmt.Errorf("listing beads: %w", err)
	}
	filtered := make([]beads.Issue, 0, len(issues))
	for _, issue := range issues {
		if strings.TrimSpace(beadType) != "" && issue.Type != beadType {
			continue
		}
		if strings.TrimSpace(status) != "" && issue.Status != status {
			continue
		}
		if strings.TrimSpace(priority) != "" && issue.Priority != priority {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.TrimSpace(cell) != "" && meta.Cell != cell {
			continue
		}
		if strings.TrimSpace(turnID) != "" && meta.TurnID != turnID {
			continue
		}
		filtered = append(filtered, issue)
	}
	printIssuesGrouped(filtered)
	return nil
}

func beadShow(home string, rest []string) error {
	if len(rest) < 2 {
		return fmt.Errorf("usage: mforge bead show <id>")
	}
	rigName, id := rest[0], rest[1]
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return fmt.Errorf("loading rig %s: %w", rigName, err)
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issue, err := client.Show(nil, id)
	if err != nil {
		return fmt.Errorf("showing bead %s: %w", id, err)
	}
	fmt.Printf("%s\n%s\nstatus=%s type=%s priority=%s\n", issue.Title, issue.ID, issue.Status, issue.Type, issue.Priority)
	fmt.Println(issue.Description)
	return nil
}

func beadClose(home string, rest []string) error {
	if len(rest) < 2 {
		return fmt.Errorf("usage: mforge bead close <id> [--reason <text>]")
	}
	rigName, id := rest[0], rest[1]
	var reason string
	for i := 2; i < len(rest); i++ {
		if rest[i] == "--reason" && i+1 < len(rest) {
			reason = rest[i+1]
			i++
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return fmt.Errorf("loading rig %s: %w", rigName, err)
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	_, err = client.Close(nil, id, reason)
	if err != nil {
		return fmt.Errorf("closing bead %s: %w", id, err)
	}
	return nil
}

func beadStatus(home string, rest []string) error {
	if len(rest) < 3 {
		return fmt.Errorf("usage: mforge bead status <id> <status> [--reason <text>]")
	}
	rigName, id, status := rest[0], rest[1], rest[2]
	var reason string
	for i := 3; i < len(rest); i++ {
		if rest[i] == "--reason" && i+1 < len(rest) {
			reason = rest[i+1]
			i++
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return fmt.Errorf("loading rig %s: %w", rigName, err)
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	if strings.EqualFold(status, "done") || strings.EqualFold(status, "closed") {
		_, err = client.Close(nil, id, reason)
		if err != nil {
			return fmt.Errorf("closing bead %s: %w", id, err)
		}
		return nil
	}
	_, err = client.UpdateStatus(nil, id, status)
	if err != nil {
		return fmt.Errorf("updating bead %s status: %w", id, err)
	}
	return nil
}

func beadTriage(home string, rest []string) error {
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge bead triage --id <id> --cell <cell> --role <role> [--turn <id>] [--promise <token>]")
	}
	rigName := rest[0]
	var id, cell, role, turnID, promise string
	promise = "DONE"
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case "--id":
			if i+1 < len(rest) {
				id = rest[i+1]
				i++
			}
		case "--cell":
			if i+1 < len(rest) {
				cell = rest[i+1]
				i++
			}
		case "--role":
			if i+1 < len(rest) {
				role = rest[i+1]
				i++
			}
		case "--turn":
			if i+1 < len(rest) {
				turnID = rest[i+1]
				i++
			}
		case "--promise":
			if i+1 < len(rest) {
				promise = rest[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(id) == "" || strings.TrimSpace(cell) == "" || strings.TrimSpace(role) == "" {
		return fmt.Errorf("--id, --cell, and --role are required")
	}
	if strings.TrimSpace(turnID) == "" {
		if state, err := turn.Load(rig.TurnStatePath(home, rigName)); err == nil {
			turnID = strings.TrimSpace(state.ID)
		}
	}
	if err := beadLimit(home, rigName, cell, turnID); err != nil {
		return fmt.Errorf("checking bead limit: %w", err)
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return fmt.Errorf("loading rig %s: %w", rigName, err)
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issue, err := client.Show(nil, id)
	if err != nil {
		return fmt.Errorf("showing bead %s: %w", id, err)
	}
	cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cell))
	if err != nil {
		return fmt.Errorf("loading cell %s: %w", cell, err)
	}
	inboxRel := fmt.Sprintf("mail/inbox/%s.md", issue.ID)
	outboxRel := fmt.Sprintf("mail/outbox/%s.md", issue.ID)
	meta := beads.Meta{
		Cell:     cell,
		Role:     role,
		Scope:    cellCfg.ScopePrefix,
		Inbox:    inboxRel,
		Outbox:   outboxRel,
		Promise:  promise,
		TurnID:   turnID,
		Worktree: cellCfg.WorktreePath,
	}
	assnDesc := beads.RenderMeta(meta) + "\n\n" + issue.Title
	assn, err := client.Create(nil, beads.CreateRequest{
		Title:       "Assignment " + issue.ID,
		Type:        "assignment",
		Priority:    issue.Priority,
		Status:      "open",
		Description: assnDesc,
		Deps:        []string{"related:" + issue.ID},
	})
	if err != nil {
		return fmt.Errorf("creating assignment: %w", err)
	}
	mail, _ := writeAssignmentInbox(cellCfg.WorktreePath, inboxRel, outboxRel, "DONE", issue)
	_ = createMailBead(client, meta, "Mail "+assn.ID, mail, []string{"related:" + assn.ID})
	_, _ = client.UpdateStatus(nil, issue.ID, "in_progress")
	fmt.Printf("Triaged bead %s -> assignment for %s/%s\n", issue.ID, cell, role)
	return nil
}

func beadDep(home string, rest []string) error {
	if len(rest) < 2 {
		return fmt.Errorf("usage: mforge bead dep add <id> <dep>")
	}
	rigName := rest[0]
	op := rest[1]
	if op != "add" {
		return fmt.Errorf("usage: mforge bead dep add <id> <dep>")
	}
	if len(rest) < 4 {
		return fmt.Errorf("usage: mforge bead dep add <id> <dep>")
	}
	id, dep := rest[2], rest[3]
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return fmt.Errorf("loading rig %s: %w", rigName, err)
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	if err := client.DepAdd(nil, id, dep); err != nil {
		return fmt.Errorf("adding dep %s to %s: %w", dep, id, err)
	}
	return nil
}

func renderTemplate(beadType, desc, acceptance, compat, links string) string {
	body := strings.TrimSpace(desc)
	needsSections := beadType == "contract" || beadType == "contractproposal" || beadType == "decision" || beadType == "migrationstep"
	if !needsSections {
		return body
	}
	lines := []string{}
	if body != "" {
		lines = append(lines, body)
	}
	if beadType == "decision" && acceptance == "" {
		acceptance = "Decision required from human/architect."
	}
	if acceptance != "" {
		lines = append(lines, "\n## Acceptance Criteria\n"+acceptance)
	}
	if compat != "" {
		lines = append(lines, "\n## Compatibility Notes\n"+compat)
	}
	if links != "" {
		lines = append(lines, "\n## Links\n"+links)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func splitCSV(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
