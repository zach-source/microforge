package subcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

type engineEvent struct {
	ID        string
	Type      string
	Scope     string
	Title     string
	Source    string
	Payload   map[string]any
	IssueType string
}

type engineCommand struct {
	Actor string
	Kind  string
	Data  map[string]string
}

func Engine(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge engine <run|emit|drain> ...")
	}
	op := args[0]
	rest := args[1:]
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge engine %s ...", op)
	}
	rigName := rest[0]
	switch op {
	case "emit":
		return engineEmit(home, rigName, rest[1:])
	case "drain":
		return engineDrain(home, rigName, rest[1:])
	case "run":
		return engineRun(home, rigName, rest[1:])
	default:
		return fmt.Errorf("unknown engine subcommand: %s", op)
	}
}

func engineEmit(home, rigName string, args []string) error {
	var eventType, scope, title, source, payloadRaw string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				eventType = args[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(args) {
				scope = args[i+1]
				i++
			}
		case "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "--source":
			if i+1 < len(args) {
				source = args[i+1]
				i++
			}
		case "--payload":
			if i+1 < len(args) {
				payloadRaw = args[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(eventType) == "" {
		return fmt.Errorf("--type is required")
	}
	payload := map[string]any{}
	if strings.TrimSpace(payloadRaw) != "" {
		if err := json.Unmarshal([]byte(payloadRaw), &payload); err != nil {
			return fmt.Errorf("invalid --payload JSON: %w", err)
		}
	}
	if strings.TrimSpace(scope) == "" {
		if v, ok := payload["scope"].(string); ok {
			scope = v
		}
	}
	if strings.TrimSpace(title) == "" {
		if v, ok := payload["title"].(string); ok {
			title = v
		}
	}
	if strings.TrimSpace(title) == "" {
		title = eventType
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	meta := beads.Meta{Kind: eventType, Scope: scope, Title: title, SourceRole: source}
	body := "{}"
	if len(payload) > 0 {
		b, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		body = string(b)
	}
	desc := beads.RenderMeta(meta) + "\n\n" + body
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       title,
		Type:        "event",
		Priority:    "p2",
		Status:      "open",
		Description: desc,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Emitted event %s (%s)\n", issue.ID, eventType)
	return nil
}

func engineDrain(home, rigName string, args []string) error {
	keep := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--keep" {
			keep = true
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return err
	}
	events := filterEventIssues(issues)
	if len(events) == 0 {
		fmt.Println("No queued events")
		return nil
	}
	for _, issue := range events {
		ev := parseEngineEvent(issue)
		fmt.Printf("%s\t%s\t%s\n", issue.ID, ev.Type, ev.Title)
		if !keep {
			_, _ = client.Close(nil, issue.ID, "event drained")
		}
	}
	return nil
}

func engineRun(home, rigName string, args []string) error {
	wait := false
	rounds := 0
	completionPromise := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--wait":
			wait = true
		case "--rounds":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil && v > 0 {
					rounds = v
				}
				i++
			}
		case "--completion-promise":
			if i+1 < len(args) {
				completionPromise = args[i+1]
				i++
			}
		}
	}
	if rounds > 0 || strings.TrimSpace(completionPromise) != "" {
		return engineRunRounds(home, rigName, rounds, completionPromise, wait)
	}
	if _, err := turn.Load(rig.TurnStatePath(home, rigName)); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := Turn(home, []string{"start", rigName}); err != nil {
			return err
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return err
	}
	intake := filterEventIssues(issues)
	fmt.Printf("Intake: %d event(s)\n", len(intake))
	planned := make([]engineCommand, 0)
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	cellByScope := buildScopeIndex(cells)
	for _, issue := range intake {
		ev := parseEngineEvent(issue)
		cmds, err := planFromEvent(home, rigName, client, cellByScope, ev)
		if err != nil {
			return err
		}
		planned = append(planned, cmds...)
		_, _ = client.Close(nil, issue.ID, "event consumed")
	}
	fmt.Printf("Plan: %d command(s)\n", len(planned))
	for _, cmd := range planned {
		switch cmd.Kind {
		case "WakeAgent":
			args := []string{"wake", rigName, cmd.Data["cell"], cmd.Data["role"]}
			_ = Agent(home, args)
		}
	}
	if wait {
		if err := Wait(home, []string{rigName}); err != nil {
			return err
		}
	}
	if _, err := reconcile(home, rigName, false); err != nil {
		return err
	}
	fmt.Println("Review: reconciled assignments and checkpointed turn state")
	return nil
}

func engineRunRounds(home, rigName string, rounds int, completionPromise string, wait bool) error {
	maxRounds := rounds
	if maxRounds <= 0 {
		maxRounds = 1000
	}
	for i := 1; i <= maxRounds; i++ {
		startArgs := []string{"start", rigName}
		if wait {
			startArgs = append(startArgs, "--wait")
		}
		if err := Round(home, startArgs); err != nil {
			return err
		}
		reviewArgs := []string{"review", rigName}
		if wait {
			reviewArgs = append(reviewArgs, "--wait")
		}
		if err := Round(home, reviewArgs); err != nil {
			return err
		}
		msg := fmt.Sprintf("engine round %d", i)
		if err := Checkpoint(home, []string{rigName, "--message", msg}); err != nil {
			fmt.Printf("Checkpoint skipped: %v\n", err)
		}
		if strings.TrimSpace(completionPromise) != "" {
			ok, err := completionPromiseMet(home, rigName, completionPromise)
			if err != nil {
				return err
			}
			if ok {
				fmt.Printf("Completion promise met: %s\n", completionPromise)
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
		if rounds > 0 && i >= rounds {
			break
		}
	}
	if strings.TrimSpace(completionPromise) != "" {
		fmt.Printf("Completion promise not met after %d round(s)\n", maxRounds)
	}
	return nil
}

func completionPromiseMet(home, rigName, token string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return false, err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return false, err
	}
	for _, issue := range issues {
		if issue.Status != "done" && issue.Status != "closed" {
			continue
		}
		if strings.Contains(issue.Title, token) || strings.Contains(issue.Description, token) {
			return true, nil
		}
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return false, err
	}
	for _, cell := range cells {
		outboxDir := filepath.Join(cell.WorktreePath, "mail", "outbox")
		entries, err := os.ReadDir(outboxDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			b, err := os.ReadFile(filepath.Join(outboxDir, entry.Name()))
			if err != nil {
				continue
			}
			if strings.Contains(string(b), token) {
				return true, nil
			}
		}
	}
	return false, nil
}

func filterEventIssues(issues []beads.Issue) []beads.Issue {
	out := make([]beads.Issue, 0)
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "event" {
			continue
		}
		if issue.Status != "open" {
			continue
		}
		out = append(out, issue)
	}
	return out
}

func parseEngineEvent(issue beads.Issue) engineEvent {
	meta := beads.ParseMeta(issue.Description)
	body := strings.TrimSpace(beads.StripMeta(issue.Description))
	payload := map[string]any{}
	if body != "" {
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			payload["raw"] = body
		}
	}
	eventType := strings.TrimSpace(meta.Kind)
	if eventType == "" {
		eventType = strings.TrimSpace(issue.Title)
	}
	scope := strings.TrimSpace(meta.Scope)
	if scope == "" {
		if v, ok := payload["scope"].(string); ok {
			scope = v
		}
	}
	title := strings.TrimSpace(meta.Title)
	if title == "" {
		if v, ok := payload["title"].(string); ok {
			title = v
		}
	}
	if title == "" {
		title = issue.Title
	}
	return engineEvent{
		ID:      issue.ID,
		Type:    eventType,
		Scope:   scope,
		Title:   title,
		Source:  meta.SourceRole,
		Payload: payload,
	}
}

func planFromEvent(home, rigName string, client beads.Client, cellByScope map[string]string, ev engineEvent) ([]engineCommand, error) {
	kind := eventKind(ev.Type)
	role := defaultRole(ev.Type)
	if v, ok := ev.Payload["role"].(string); ok && strings.TrimSpace(v) != "" {
		role = v
	}
	cell := ""
	if v, ok := ev.Payload["cell"].(string); ok {
		cell = v
	}
	if cell == "" {
		cell = pickCellForScope(cellByScope, ev.Scope)
	}
	if cell == "" {
		return nil, fmt.Errorf("no cell match for scope %q", ev.Scope)
	}
	meta := beads.Meta{Scope: ev.Scope, Kind: kind, Title: ev.Title}
	desc := beads.RenderMeta(meta)
	if raw, ok := ev.Payload["raw"].(string); ok && strings.TrimSpace(raw) != "" {
		desc += "\n\n" + raw
	}
	issue, err := client.Create(nil, beads.CreateRequest{
		Title:       ev.Title,
		Type:        "task",
		Priority:    "p2",
		Status:      "open",
		Description: desc,
		Deps:        []string{"related:" + ev.ID},
	})
	if err != nil {
		return nil, err
	}
	if err := Assign(home, []string{rigName, "--task", issue.ID, "--cell", cell, "--role", role}); err != nil {
		return nil, err
	}
	cmd := engineCommand{
		Actor: "MGR",
		Kind:  "WakeAgent",
		Data: map[string]string{
			"cell": cell,
			"role": role,
		},
	}
	return []engineCommand{cmd}, nil
}

func eventKind(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "featurerequest":
		return "improve"
	case "reviewrequest":
		return "fix"
	case "monitorrequest":
		return "monitor"
	case "docupdateneeded":
		return "doc"
	default:
		return "task"
	}
}

func defaultRole(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "docupdateneeded":
		return "architect"
	default:
		return "builder"
	}
}

func buildScopeIndex(cells []rig.CellConfig) map[string]string {
	index := map[string]string{}
	for _, cell := range cells {
		scope := strings.TrimSpace(cell.ScopePrefix)
		if scope == "" {
			continue
		}
		index[scope] = cell.Name
	}
	return index
}

func pickCellForScope(index map[string]string, scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return ""
	}
	keys := make([]string, 0, len(index))
	for k := range index {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, prefix := range keys {
		if strings.HasPrefix(scope, prefix) {
			return index[prefix]
		}
	}
	return ""
}
