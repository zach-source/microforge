package beads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/util"
)

type Client struct {
	RepoPath string
}

type DeleteOptions struct {
	Force   bool
	Hard    bool
	Cascade bool
	DryRun  bool
	Reason  string
}

type CloseOptions struct {
	Force  bool
	Reason string
}

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

type CreateRequest struct {
	Title       string
	Type        string
	Priority    string
	Status      string
	Description string
	Deps        []string
}

var ErrUpdateDescriptionUnsupported = errors.New("bd update does not support --description")

func (c Client) Init(ctx context.Context) error {
	_, err := util.RunInDir(ctx, c.RepoPath, "bd", "init")
	return err
}

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
		return Issue{}, err
	}
	issue, err := parseIssue(res.Stdout)
	if err != nil {
		return Issue{}, err
	}
	if strings.TrimSpace(req.Status) != "" {
		updated, err := c.UpdateStatus(ctx, issue.ID, req.Status)
		if err != nil {
			return Issue{}, err
		}
		return updated, nil
	}
	return issue, nil
}

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
		return Issue{}, err
	}
	return parseIssue(res.Stdout)
}

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
		return Issue{}, err
	}
	return parseIssue(res.Stdout)
}

func (c Client) Close(ctx context.Context, id, reason string) (Issue, error) {
	return c.CloseWithOptions(ctx, id, CloseOptions{Reason: reason})
}

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
		return Issue{}, err
	}
	return parseIssue(res.Stdout)
}

func (c Client) Show(ctx context.Context, id string) (Issue, error) {
	if strings.TrimSpace(id) == "" {
		return Issue{}, fmt.Errorf("id is required")
	}
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", "show", id, "--json")
	if err != nil {
		return Issue{}, err
	}
	return parseIssue(res.Stdout)
}

func (c Client) List(ctx context.Context) ([]Issue, error) {
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", "list", "--json")
	if err != nil {
		return nil, err
	}
	return parseIssues(res.Stdout)
}

func (c Client) Ready(ctx context.Context) ([]Issue, error) {
	res, err := util.RunInDir(ctx, c.RepoPath, "bd", "ready", "--json")
	if err != nil {
		return nil, err
	}
	return parseIssues(res.Stdout)
}

func (c Client) DepAdd(ctx context.Context, id string, dep string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(dep) == "" {
		return fmt.Errorf("id and dep required")
	}
	_, err := util.RunInDir(ctx, c.RepoPath, "bd", "dep", "add", id, dep)
	return err
}

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
		return "", err
	}
	return res.Stdout, nil
}

func parseIssue(raw string) (Issue, error) {
	issues, err := parseIssues(raw)
	if err != nil {
		return Issue{}, err
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
		return nil, err
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
