package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

type reconcileSummary struct {
	AssignmentsClosed int
	TasksUnblocked    int
	AgentsStale       int
	AgentsIdle        int
	AgentsDown        int
}

func Manager(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge manager <tick|assign> <rig> [--watch] [--role <role>]")
	}
	op := args[0]
	rest := args[1:]
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge manager %s <rig> [--watch] [--role <role>]", op)
	}
	rigName := rest[0]
	if op == "assign" {
		return ManagerAssign(home, rigName, rest[1:])
	}
	if op != "tick" {
		return fmt.Errorf("unknown manager subcommand: %s", op)
	}
	watch := false
	for i := 1; i < len(rest); i++ {
		if rest[i] == "--watch" {
			watch = true
		}
	}
	for {
		summary, err := reconcile(home, rigName)
		if err != nil {
			return err
		}
		if summary.AssignmentsClosed > 0 {
			fmt.Printf("Reconciled %d assignment(s) to done\n", summary.AssignmentsClosed)
		}
		if summary.TasksUnblocked > 0 {
			fmt.Printf("Unblocked %d task(s)\n", summary.TasksUnblocked)
		}
		if summary.AgentsDown > 0 || summary.AgentsStale > 0 || summary.AgentsIdle > 0 {
			fmt.Printf("Agent health: down=%d stale=%d idle=%d\n", summary.AgentsDown, summary.AgentsStale, summary.AgentsIdle)
		}
		if !watch {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

func reconcile(home, rigName string) (reconcileSummary, error) {
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return reconcileSummary{}, err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return reconcileSummary{}, err
	}
	eventGate := eventKinds(issues)
	summary := reconcileSummary{}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "assignment" {
			continue
		}
		if issue.Status != "in_progress" && issue.Status != "open" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.TrimSpace(meta.Worktree) == "" || strings.TrimSpace(meta.Outbox) == "" || strings.TrimSpace(meta.Promise) == "" {
			continue
		}
		outAbs := filepath.Join(meta.Worktree, meta.Outbox)
		b, err := os.ReadFile(outAbs)
		if err != nil {
			continue
		}
		if strings.Contains(string(b), meta.Promise) {
			if ok, err := assignmentHasCommit(meta.Worktree, issue); err == nil && !ok {
				key := "assignment_missing_commit|" + issue.ID
				if !eventGate[key] {
					meta.Kind = "assignment_missing_commit"
					meta.Title = issue.Title
					emitOrchestrationEvent(cfg.RepoPath, meta, fmt.Sprintf("Assignment missing commit %s", issue.ID), []string{"related:" + issue.ID})
					eventGate[key] = true
				}
				continue
			}
			_, _ = client.Close(nil, issue.ID, "assignment complete")
			archiveMail(meta.Worktree, meta.Inbox)
			archiveMail(meta.Worktree, meta.Outbox)
			summary.AssignmentsClosed++
			meta.Kind = "assignment_complete"
			meta.Title = issue.Title
			emitOrchestrationEvent(cfg.RepoPath, meta, fmt.Sprintf("Assignment complete %s", issue.ID), []string{"related:" + issue.ID})
		}
	}
	unblocked, err := reconcileBlockedTasks(client, issues, cfg.RepoPath)
	if err != nil {
		return summary, err
	}
	summary.TasksUnblocked = unblocked
	health, err := reconcileAgentHealth(home, cfg, rigName, issues)
	if err != nil {
		return summary, err
	}
	summary.AgentsStale = health.Stale
	summary.AgentsIdle = health.Idle
	summary.AgentsDown = health.Down
	return summary, nil
}

func archiveMail(worktree, rel string) {
	if strings.TrimSpace(rel) == "" {
		return
	}
	src := filepath.Join(worktree, rel)
	dst := filepath.Join(worktree, "mail", "archive", filepath.Base(rel))
	_ = os.MkdirAll(filepath.Dir(dst), 0o755)
	_ = os.Rename(src, dst)
}

type agentHealthSummary struct {
	Stale int
	Idle  int
	Down  int
}

func reconcileAgentHealth(home string, cfg rig.RigConfig, rigName string, issues []beads.Issue) (agentHealthSummary, error) {
	const staleThreshold = 15 * time.Minute
	const idleThreshold = 5 * time.Minute
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return agentHealthSummary{}, err
	}
	roles := []string{"builder", "monitor", "reviewer", "architect", "cell"}
	eventGate := map[string]bool{}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "event" {
			continue
		}
		if issue.Status == "closed" || issue.Status == "done" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		key := strings.TrimSpace(meta.Kind) + "|" + meta.Cell + "|" + meta.Role
		if key != "||" {
			eventGate[key] = true
		}
	}
	now := time.Now().UTC()
	summary := agentHealthSummary{}
	for _, cell := range cells {
		for _, role := range roles {
			hb := readHeartbeat(agentObsDir(home, rigName, cell.Name, role))
			if strings.TrimSpace(hb.Timestamp) == "" {
				continue
			}
			ts, err := time.Parse(time.RFC3339, hb.Timestamp)
			if err != nil {
				continue
			}
			age := now.Sub(ts)
			session := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, cell.Name, role)
			_, sessErr := runTmux(cfg, false, false, "has-session", "-t", session)
			running := sessErr == nil
			meta := beads.Meta{
				Cell:  cell.Name,
				Role:  role,
				Scope: cell.ScopePrefix,
				Kind:  "",
			}
			if !running {
				if age > staleThreshold {
					meta.Kind = "agent_down"
					if !eventGate[meta.Kind+"|"+cell.Name+"|"+role] {
						emitOrchestrationEvent(cfg.RepoPath, meta, fmt.Sprintf("Agent down %s/%s", cell.Name, role), nil)
						eventGate[meta.Kind+"|"+cell.Name+"|"+role] = true
					}
					summary.Down++
				}
				continue
			}
			if strings.EqualFold(hb.Status, "idle") {
				if age > idleThreshold {
					meta.Kind = "agent_idle"
					if !eventGate[meta.Kind+"|"+cell.Name+"|"+role] {
						emitOrchestrationEvent(cfg.RepoPath, meta, fmt.Sprintf("Agent idle %s/%s", cell.Name, role), nil)
						eventGate[meta.Kind+"|"+cell.Name+"|"+role] = true
					}
					summary.Idle++
				}
				continue
			}
			if age > staleThreshold {
				meta.Kind = "agent_stale"
				if !eventGate[meta.Kind+"|"+cell.Name+"|"+role] {
					emitOrchestrationEvent(cfg.RepoPath, meta, fmt.Sprintf("Agent stale %s/%s", cell.Name, role), nil)
					eventGate[meta.Kind+"|"+cell.Name+"|"+role] = true
				}
				summary.Stale++
			}
		}
	}
	return summary, nil
}

func reconcileBlockedTasks(client beads.Client, issues []beads.Issue, repo string) (int, error) {
	issueByID := map[string]beads.Issue{}
	for _, issue := range issues {
		issueByID[issue.ID] = issue
	}
	unblocked := 0
	for _, issue := range issues {
		if strings.ToLower(issue.Status) != "blocked" {
			continue
		}
		if strings.ToLower(issue.Type) == "assignment" {
			continue
		}
		if len(issue.Deps) == 0 {
			continue
		}
		if !depsSatisfied(issue.Deps, issueByID) {
			continue
		}
		if _, err := client.UpdateStatus(nil, issue.ID, "open"); err != nil {
			return unblocked, err
		}
		unblocked++
		meta := beads.ParseMeta(issue.Description)
		meta.Kind = "task_unblocked"
		meta.Title = issue.Title
		emitOrchestrationEvent(repo, meta, fmt.Sprintf("Task unblocked %s", issue.ID), []string{"related:" + issue.ID})
	}
	return unblocked, nil
}

func depsSatisfied(deps []string, issues map[string]beads.Issue) bool {
	for _, dep := range deps {
		id := depID(dep)
		if strings.TrimSpace(id) == "" {
			continue
		}
		issue, ok := issues[id]
		if !ok {
			return false
		}
		status := strings.ToLower(strings.TrimSpace(issue.Status))
		if status != "done" && status != "closed" {
			return false
		}
	}
	return true
}

func depID(dep string) string {
	if strings.Contains(dep, ":") {
		parts := strings.SplitN(dep, ":", 2)
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(dep)
}

func eventKinds(issues []beads.Issue) map[string]bool {
	out := map[string]bool{}
	for _, issue := range issues {
		if strings.ToLower(issue.Type) != "event" {
			continue
		}
		if issue.Status == "closed" || issue.Status == "done" {
			continue
		}
		meta := beads.ParseMeta(issue.Description)
		if strings.TrimSpace(meta.Kind) == "" {
			continue
		}
		key := meta.Kind
		if strings.TrimSpace(meta.Cell) != "" || strings.TrimSpace(meta.Role) != "" {
			key = key + "|" + meta.Cell + "|" + meta.Role
		}
		out[key] = true
	}
	return out
}

func assignmentHasCommit(worktree string, issue beads.Issue) (bool, error) {
	if strings.TrimSpace(worktree) == "" {
		return true, nil
	}
	if _, err := os.Stat(worktree); err != nil {
		return false, err
	}
	if _, err := util.Run(nil, "git", "-C", worktree, "rev-parse", "--is-inside-work-tree"); err != nil {
		return true, nil
	}
	res, err := util.Run(nil, "git", "-C", worktree, "log", "-1", "--pretty=%B")
	if err != nil {
		return false, err
	}
	msg := strings.ToLower(res.Stdout)
	if msg == "" {
		return false, nil
	}
	if strings.Contains(msg, strings.ToLower(issue.ID)) {
		return true, nil
	}
	for _, dep := range issue.Deps {
		if strings.HasPrefix(dep, "related:") {
			taskID := strings.TrimPrefix(dep, "related:")
			if strings.TrimSpace(taskID) != "" && strings.Contains(msg, strings.ToLower(taskID)) {
				return true, nil
			}
		}
	}
	return false, nil
}
