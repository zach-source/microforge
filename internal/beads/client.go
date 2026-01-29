// Package beads provides a client wrapper for the bd (Beads) CLI issue tracker.
// Beads is the task/request/observation tracking system used by Microforge agents.
package beads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/util"
)

// Client wraps the bd CLI for issue operations. RepoPath is the directory
// where bd commands are executed (typically the monorepo root).
type Client struct {
	RepoPath string
}

// DeleteOptions controls the behavior of the Delete operation.
type DeleteOptions struct {
	Force   bool
	Hard    bool
	Cascade bool
	DryRun  bool
	Reason  string
}

// CloseOptions controls the behavior of the Close operation.
type CloseOptions struct {
	Force  bool
	Reason string
}

// Issue represents a bead (task, assignment, event, etc.) in the tracker.
type Issue struct {
	ID          string
	Title       string
	Status      string
	Type        string
	Priority    string
	Description string
	Deps        []string
	CreatedAt   string
}

// CreateRequest contains fields for creating a new bead.
type CreateRequest struct {
	Title       string
	Type        string
	Priority    string
	Status      string
	Description string
	Deps        []string
}

// ErrUpdateDescriptionUnsupported is returned when the bd CLI doesn't support --description flag.
var ErrUpdateDescriptionUnsupported = errors.New("bd update does not support --description")

// Init initializes the beads store in the repository.
func (c Client) Init(ctx context.Context) error {
	_, err := util.RunInDir(ctx, c.RepoPath, "bd", "init")
	if err != nil {
		return fmt.Errorf("bd init: %w", err)
	}
	return nil
}

// Create creates a new bead with the given parameters.
func (c Client) Create(ctx context.Context, req CreateRequest) (Issue, error) {
	if strings.TrimSpace(req.Title) == "" {
		return Issue{}, fmt.Errorf("title is required")
	}
	args := []string{"create", "--json"}
	if strings.TrimSpace(req.Type) != "" {
		args = append(args, "-t", req.Type)
	}
	if strings.TrimSpace(req.Priority) != "" {
		args = append(args, "-p", req.Priority)
	}
	if strings.TrimSpace(req.Description) != "" {
		args = append(args, "--description", req.Description)
	}
	for _, dep := range req.Deps {
		if strings.TrimSpace(dep) == "" {
			continue
		}
		args = append(args, "--deps", dep)
	}
	args = append(args, req.Title)
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", args...)
	if err != nil {
		return Issue{}, fmt.Errorf("bd create: %w", err)
	}
	issue, err := parseIssue(res.Stdout)
	if err != nil {
		return Issue{}, fmt.Errorf("parsing bd create response: %w", err)
	}
	if strings.TrimSpace(req.Status) != "" {
		updated, err := c.UpdateStatus(ctx, issue.ID, req.Status)
		if err != nil {
			return Issue{}, fmt.Errorf("setting status on %s: %w", issue.ID, err)
		}
		return updated, nil
	}
	return issue, nil
}

// UpdateStatus changes the status of an existing bead.
func (c Client) UpdateStatus(ctx context.Context, id, status string) (Issue, error) {
	if strings.TrimSpace(id) == "" {
		return Issue{}, fmt.Errorf("id is required")
	}
	args := []string{"update", id, "--json"}
	if strings.TrimSpace(status) != "" {
		args = append(args, "--status", status)
	}
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", args...)
	if err != nil {
		return Issue{}, fmt.Errorf("bd update %s: %w", id, err)
	}
	issue, err := parseIssue(res.Stdout)
	if err != nil {
		return Issue{}, fmt.Errorf("parsing bd update response: %w", err)
	}
	return issue, nil
}

// UpdateDescription updates the description of an existing bead.
// Returns ErrUpdateDescriptionUnsupported if the bd CLI doesn't support this operation.
func (c Client) UpdateDescription(ctx context.Context, id, description string) (Issue, error) {
	if strings.TrimSpace(id) == "" {
		return Issue{}, fmt.Errorf("id is required")
	}
	args := []string{"update", id, "--json"}
	if strings.TrimSpace(description) != "" {
		args = append(args, "--description", description)
	}
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", args...)
	if err != nil {
		if strings.Contains(err.Error(), "unknown flag") && strings.Contains(err.Error(), "--description") {
			return Issue{}, ErrUpdateDescriptionUnsupported
		}
		return Issue{}, fmt.Errorf("bd update %s description: %w", id, err)
	}
	issue, err := parseIssue(res.Stdout)
	if err != nil {
		return Issue{}, fmt.Errorf("parsing bd update response: %w", err)
	}
	return issue, nil
}

// Close closes a bead with an optional reason.
func (c Client) Close(ctx context.Context, id, reason string) (Issue, error) {
	return c.CloseWithOptions(ctx, id, CloseOptions{Reason: reason})
}

// CloseWithOptions closes a bead with configurable options.
func (c Client) CloseWithOptions(ctx context.Context, id string, opts CloseOptions) (Issue, error) {
	if strings.TrimSpace(id) == "" {
		return Issue{}, fmt.Errorf("id is required")
	}
	args := []string{"close", id, "--json"}
	if opts.Force {
		args = append(args, "--force")
	}
	if strings.TrimSpace(opts.Reason) != "" {
		args = append(args, "--reason", opts.Reason)
	}
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", args...)
	if err != nil {
		return Issue{}, fmt.Errorf("bd close %s: %w", id, err)
	}
	issue, err := parseIssue(res.Stdout)
	if err != nil {
		return Issue{}, fmt.Errorf("parsing bd close response: %w", err)
	}
	return issue, nil
}

// Show retrieves a single bead by ID.
func (c Client) Show(ctx context.Context, id string) (Issue, error) {
	if strings.TrimSpace(id) == "" {
		return Issue{}, fmt.Errorf("id is required")
	}
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", "show", id, "--json")
	if err != nil {
		return Issue{}, fmt.Errorf("bd show %s: %w", id, err)
	}
	issue, err := parseIssue(res.Stdout)
	if err != nil {
		return Issue{}, fmt.Errorf("parsing bd show response: %w", err)
	}
	return issue, nil
}

// List returns all beads in the repository.
func (c Client) List(ctx context.Context) ([]Issue, error) {
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", "list", "--json")
	if err != nil {
		return nil, fmt.Errorf("bd list: %w", err)
	}
	issues, err := parseIssues(res.Stdout)
	if err != nil {
		return nil, fmt.Errorf("parsing bd list response: %w", err)
	}
	return issues, nil
}

// Ready returns beads that are ready to be worked on (no blocking dependencies).
func (c Client) Ready(ctx context.Context) ([]Issue, error) {
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", "ready", "--json")
	if err != nil {
		return nil, fmt.Errorf("bd ready: %w", err)
	}
	issues, err := parseIssues(res.Stdout)
	if err != nil {
		return nil, fmt.Errorf("parsing bd ready response: %w", err)
	}
	return issues, nil
}

// DepAdd adds a dependency relationship between beads.
func (c Client) DepAdd(ctx context.Context, id string, dep string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(dep) == "" {
		return fmt.Errorf("id and dep required")
	}
	_, err := util.RunInDir(ctx, c.RepoPath, "bd", "dep", "add", id, dep)
	if err != nil {
		return fmt.Errorf("bd dep add %s %s: %w", id, dep, err)
	}
	return nil
}

// Delete removes beads with the given IDs. Use DryRun to preview the operation.
func (c Client) Delete(ctx context.Context, ids []string, opts DeleteOptions) (string, error) {
	if len(ids) == 0 {
		return "", fmt.Errorf("id is required")
	}
	args := []string{"delete"}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			continue
		}
		args = append(args, id)
	}
	if len(args) == 1 {
		return "", fmt.Errorf("id is required")
	}
	if opts.Cascade {
		args = append(args, "--cascade")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.Hard {
		args = append(args, "--hard")
	}
	if opts.Force {
		args = append(args, "--force")
	}
	if strings.TrimSpace(opts.Reason) != "" {
		args = append(args, "--reason", opts.Reason)
	}
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", args...)
	if err != nil {
		return "", fmt.Errorf("bd delete: %w", err)
	}
	return res.Stdout, nil
}

func parseIssue(raw string) (Issue, error) {
	issues, err := parseIssues(raw)
	if err != nil {
		return Issue{}, fmt.Errorf("parsing issue: %w", err)
	}
	if len(issues) == 1 {
		return issues[0], nil
	}
	if len(issues) > 1 {
		return issues[0], nil
	}
	return Issue{}, fmt.Errorf("no issue in response")
}

func parseIssues(raw string) ([]Issue, error) {
	trim := strings.TrimSpace(raw)
	if trim == "" {
		return nil, fmt.Errorf("empty response")
	}
	var anyVal any
	if err := json.Unmarshal([]byte(trim), &anyVal); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	switch v := anyVal.(type) {
	case []any:
		return decodeIssueArray(v), nil
	case map[string]any:
		if issuesRaw, ok := v["issues"]; ok {
			if arr, ok := issuesRaw.([]any); ok {
				return decodeIssueArray(arr), nil
			}
		}
		return []Issue{decodeIssueMap(v)}, nil
	default:
		return nil, fmt.Errorf("unexpected json format")
	}
}

func decodeIssueArray(arr []any) []Issue {
	out := make([]Issue, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, decodeIssueMap(m))
	}
	return out
}

func decodeIssueMap(m map[string]any) Issue {
	issue := Issue{}
	if v, ok := m["id"].(string); ok {
		issue.ID = v
	}
	if v, ok := m["title"].(string); ok {
		issue.Title = v
	}
	if v, ok := m["status"].(string); ok {
		issue.Status = v
	}
	if v, ok := m["type"].(string); ok {
		issue.Type = v
	}
	if issue.Type == "" {
		if v, ok := m["issue_type"].(string); ok {
			issue.Type = v
		}
	}
	if v, ok := m["priority"].(string); ok {
		issue.Priority = v
	}
	if issue.Priority == "" {
		switch v := m["priority"].(type) {
		case float64:
			if v > 0 {
				issue.Priority = fmt.Sprintf("p%d", int(v))
			}
		case int:
			if v > 0 {
				issue.Priority = fmt.Sprintf("p%d", v)
			}
		case json.Number:
			if n, err := v.Int64(); err == nil && n > 0 {
				issue.Priority = fmt.Sprintf("p%d", n)
			}
		}
	}
	if v, ok := m["description"].(string); ok {
		issue.Description = v
	}
	if v, ok := m["created_at"].(string); ok {
		issue.CreatedAt = v
	}
	if v, ok := m["created"].(string); ok && issue.CreatedAt == "" {
		issue.CreatedAt = v
	}
	if depsRaw, ok := m["deps"].([]any); ok {
		for _, depAny := range depsRaw {
			if depStr, ok := depAny.(string); ok {
				issue.Deps = append(issue.Deps, depStr)
			}
		}
	}
	return issue
}
