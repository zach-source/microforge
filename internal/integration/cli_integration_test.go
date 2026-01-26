package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/cli"
	"github.com/example/microforge/internal/rig"
)

type fakeIssue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Type        string   `json:"type"`
	Priority    string   `json:"priority"`
	Description string   `json:"description"`
	Deps        []string `json:"deps"`
	CreatedAt   string   `json:"created_at"`
}

type fakeStore struct {
	NextID int         `json:"next_id"`
	Issues []fakeIssue `json:"issues"`
}

func TestMain(m *testing.M) {
	if os.Getenv("MF_TEST_MODE") == "cli" {
		runCLIFromEnv()
		return
	}
	switch filepath.Base(os.Args[0]) {
	case "bd":
		runFakeBD()
		return
	case "tmux":
		runFakeTmux()
		return
	case "ssh":
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}

	storePath := filepath.Join(tmp, "bd_store.json")
	tmuxStore := filepath.Join(tmp, "tmux_store.json")

	setupFakeEnv(t, tmp, storePath, tmuxStore)

	rigName := "demo"
	runCLI(t, "init", rigName, "--repo", repo)
	runCLI(t, "context", "set", rigName)
	runCLI(t, "cell", "add", "alpha", "--scope", "apps/alpha")
	runCLI(t, "cell", "add", "beta", "--scope", "apps/beta")
	runCLI(t, "cell", "bootstrap", "alpha")
	runCLI(t, "cell", "bootstrap", "beta")
	runCLI(t, "cell", "agent-file", "alpha", "--role", "builder")
	runCLI(t, "agent", "spawn", "alpha", "builder")
	runCLI(t, "agent", "status")
	runCLI(t, "agent", "attach", "alpha", "builder")
	runCLI(t, "agent", "wake", "alpha", "builder")
	runCLI(t, "agent", "stop", "alpha", "builder")

	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		t.Fatal(err)
	}
	cfg.RemoteHost = "example.com"
	if err := rig.SaveRigConfig(rig.RigConfigPath(home, rigName), cfg); err != nil {
		t.Fatal(err)
	}

	runCLI(t, "turn", "start")
	runCLI(t, "turn", "status")

	runCLI(t, "task", "create", "--title", "Build dashboard", "--scope", "apps/alpha", "--kind", "improve")
	primaryTask := latestIssueByType(t, storePath, "task")
	runCLI(t, "task", "list")

	runCLI(t, "task", "create", "--title", "Global task")
	globalTask := latestIssueByType(t, storePath, "task")
	runCLI(t, "task", "update", "--task", globalTask.ID, "--scope", "apps/beta")

	runCLI(t, "task", "create", "--title", "Review dashboard", "--scope", "apps/alpha", "--kind", "review")
	reviewTask := latestIssueByType(t, storePath, "task")

	runCLI(t, "epic", "create", "--title", "Dashboard epic")
	epic := latestIssueByType(t, storePath, "epic")
	runCLI(t, "epic", "add-task", "--epic", epic.ID, "--task", reviewTask.ID)
	runCLI(t, "bead", "close", reviewTask.ID)
	runCLI(t, "epic", "status", "--epic", epic.ID)
	runCLI(t, "epic", "assign", "--epic", epic.ID, "--role", "builder")
	runCLI(t, "epic", "conflict", "--epic", epic.ID, "--cell", "alpha", "--details", "conflict detail")
	runCLI(t, "epic", "close", "--epic", epic.ID)

	payload := `{"title":"Need metrics","body":"Add charts","kind":"improve","scope":"apps/alpha"}`
	runCLI(t, "request", "create", "--cell", "alpha", "--role", "builder", "--severity", "med", "--priority", "p2", "--scope", "apps/alpha", "--payload", payload)
	requestIssue := latestIssueByType(t, storePath, "request")
	runCLI(t, "request", "list")
	runCLI(t, "request", "triage", "--request", requestIssue.ID, "--action", "create-task")

	runCLI(t, "assign", "--task", primaryTask.ID, "--cell", "alpha", "--role", "builder", "--promise", "DONE-TASK")

	runCLI(t, "monitor", "run-tests", "alpha", "--cmd", "true", "--severity", "high", "--priority", "p1", "--scope", "apps/alpha")
	runCLI(t, "monitor", "run", "alpha", "--cmd", "false", "--severity", "low", "--priority", "p3", "--scope", "apps/alpha", "--observation", "Signal check")

	runCLI(t, "bead", "create", "--type", "decision", "--title", "Decide metrics", "--cell", "alpha", "--scope", "apps/alpha", "--severity", "low", "--description", "decide", "--acceptance", "ok", "--compat", "none", "--links", "http://example", "--status", "open")
	decision := latestIssueByType(t, storePath, "decision")
	runCLI(t, "bead", "show", decision.ID)
	runCLI(t, "bead", "list")
	runCLI(t, "bead", "close", decision.ID, "--reason", "done")

	runCLI(t, "bead", "create", "--type", "task", "--title", "Extra task", "--cell", "alpha", "--scope", "apps/alpha")
	extraTask := latestIssueByType(t, storePath, "task")
	runCLI(t, "bead", "triage", "--id", extraTask.ID, "--cell", "alpha", "--role", "builder", "--promise", "DONE-EXTRA")
	runCLI(t, "bead", "dep", "add", extraTask.ID, "related:"+primaryTask.ID)
	runCLI(t, "bead", "template", "--type", "contract", "--title", "Contract template", "--cell", "alpha", "--scope", "apps/alpha")

	runCLI(t, "review", "create", "--title", "Review PR", "--cell", "alpha", "--scope", "apps/alpha")
	reviewIssue := latestIssueByType(t, storePath, "review")
	runCLI(t, "bead", "close", reviewIssue.ID)

	runCLI(t, "pr", "create", "--title", "Add dashboard", "--cell", "alpha", "--url", "https://example/pr/1")
	prIssue := latestIssueByType(t, storePath, "pr")
	runCLI(t, "pr", "ready", prIssue.ID)
	runCLI(t, "pr", "link-review", prIssue.ID, reviewIssue.ID)

	runCLI(t, "merge", "run", "--as", "merge-manager", "--dry-run")
	runCLI(t, "coordinator", "sync")
	runCLI(t, "digest", "render")

	runCLI(t, "build", "record", "--service", "api", "--image", "v1")
	runCLI(t, "deploy", "record", "--env", "prod", "--service", "api")
	runCLI(t, "contract", "create", "--title", "API contract", "--cell", "alpha", "--scope", "apps/alpha", "--acceptance", "- [ ] ok", "--compat", "none", "--links", "link")
	runCLI(t, "architect", "docs", "--cell", "alpha", "--details", "Update docs", "--scope", "apps/alpha")
	runCLI(t, "architect", "contract", "--cell", "alpha", "--details", "Check contract", "--scope", "apps/alpha")
	runCLI(t, "architect", "design", "--cell", "alpha", "--details", "Review design", "--scope", "apps/alpha")

	runCLI(t, "task", "split", "--task", primaryTask.ID, "--cells", "alpha,beta")

	runCLI(t, "manager", "assign", "--role", "builder")
	markAssignmentComplete(t, storePath)
	runCLI(t, "manager", "tick")

	runCLI(t, "turn", "slate")
	runCLI(t, "wait", "--turn", "no-turn", "--interval", "1")

	runCLI(t, "turn", "run", "--role", "builder")
	runCLI(t, "turn", "end")

	runCLI(t, "bead", "list", "--type", "task")
	runCLI(t, "report")

	runLibraryQuery(t, rigName)
	runLibraryStartSmoke(t, rigName)

	runCLI(t, "ssh", "--cmd", "echo", "hello")

	worktree := rig.CellWorktreeDir(home, rigName, "alpha")
	runHookWithInput(t, worktree, `{"tool_name":"Write","tool_input":{"path":"apps/alpha/file.go"}}`)
	runHookWithInput(t, worktree, `{"hook_event_name":"Stop"}`)
}

func setupFakeEnv(t *testing.T, tmp, storePath, tmuxStore string) {
	t.Helper()
	fakeBin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bd", "tmux", "ssh"} {
		path := filepath.Join(fakeBin, name)
		if err := os.Symlink(exe, path); err != nil {
			t.Fatal(err)
		}
	}
	pathEnv := fakeBin + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", pathEnv)
	t.Setenv("MF_HOME", filepath.Join(tmp, "home"))
	t.Setenv("MF_BEAD_LIMIT_PER_TURN", "1000")
	t.Setenv("MF_FAKE_BD_STORE", storePath)
	t.Setenv("MF_FAKE_TMUX_STORE", tmuxStore)
}

func runCLI(t *testing.T, args ...string) {
	t.Helper()
	if err := cli.Run(args); err != nil {
		t.Fatalf("cli %v failed: %v", args, err)
	}
}

func runHookWithInput(t *testing.T, cwd, input string) {
	t.Helper()
	origWD, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWD)
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(input))
	_ = w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = origStdin
		_ = r.Close()
	}()

	runCLI(t, "hook", "guardrails")
	runCLI(t, "hook", "stop")
}

func runLibraryQuery(t *testing.T, rigName string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&struct{}{})
		resp := map[string]any{
			"results": []map[string]string{{"service": "docs", "path": "README.md", "snippet": "ok"}},
			"source":  "local",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	runCLI(t, "library", "query", "--q", "dashboard", "--addr", addr)
}

func runLibraryStartSmoke(t *testing.T, rigName string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	args := []string{"library", "start", "--addr", "127.0.0.1:0"}
	cmd := exec.CommandContext(ctx, exe)
	cmd.Env = append(os.Environ(),
		"MF_TEST_MODE=cli",
		"MF_TEST_CLI_ARGS="+strings.Join(args, "\x1f"),
	)
	_ = cmd.Run()
}

func markAssignmentComplete(t *testing.T, storePath string) {
	t.Helper()
	issue := latestIssueByType(t, storePath, "assignment")
	meta := beads.ParseMeta(issue.Description)
	if meta.Worktree == "" || meta.Outbox == "" || meta.Promise == "" {
		t.Fatalf("assignment missing meta: %+v", meta)
	}
	outPath := filepath.Join(meta.Worktree, meta.Outbox)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, []byte(meta.Promise), 0o644); err != nil {
		t.Fatal(err)
	}
}

func latestIssueByType(t *testing.T, storePath, issueType string) fakeIssue {
	t.Helper()
	store := loadStore(t, storePath)
	for i := len(store.Issues) - 1; i >= 0; i-- {
		if strings.EqualFold(store.Issues[i].Type, issueType) {
			return store.Issues[i]
		}
	}
	t.Fatalf("no issue of type %s", issueType)
	return fakeIssue{}
}

func loadStore(t *testing.T, storePath string) fakeStore {
	t.Helper()
	b, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fakeStore{NextID: 1}
		}
		t.Fatal(err)
	}
	var store fakeStore
	if err := json.Unmarshal(b, &store); err != nil {
		t.Fatal(err)
	}
	if store.NextID == 0 {
		store.NextID = 1
	}
	return store
}

func runCLIFromEnv() {
	argsEnv := os.Getenv("MF_TEST_CLI_ARGS")
	if argsEnv == "" {
		fmt.Fprintln(os.Stderr, "missing MF_TEST_CLI_ARGS")
		os.Exit(2)
	}
	args := strings.Split(argsEnv, "\x1f")
	if err := cli.Run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runFakeBD() {
	storePath := os.Getenv("MF_FAKE_BD_STORE")
	if storePath == "" {
		fmt.Fprintln(os.Stderr, "missing MF_FAKE_BD_STORE")
		os.Exit(2)
	}
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing bd subcommand")
		os.Exit(2)
	}
	switch args[0] {
	case "init":
		_ = os.MkdirAll(filepath.Join(mustGetwd(), ".beads"), 0o755)
		_ = saveStore(storePath, loadStoreRaw(storePath))
		os.Exit(0)
	case "create":
		issue := handleBdCreate(storePath, args[1:])
		_ = json.NewEncoder(os.Stdout).Encode(issue)
		os.Exit(0)
	case "update":
		issue := handleBdUpdate(storePath, args[1:])
		_ = json.NewEncoder(os.Stdout).Encode(issue)
		os.Exit(0)
	case "close":
		issue := handleBdClose(storePath, args[1:])
		_ = json.NewEncoder(os.Stdout).Encode(issue)
		os.Exit(0)
	case "show":
		issue := handleBdShow(storePath, args[1:])
		_ = json.NewEncoder(os.Stdout).Encode(issue)
		os.Exit(0)
	case "list":
		store := loadStoreRaw(storePath)
		_ = json.NewEncoder(os.Stdout).Encode(store.Issues)
		os.Exit(0)
	case "ready":
		store := loadStoreRaw(storePath)
		ready := []fakeIssue{}
		for _, issue := range store.Issues {
			if issue.Status == "ready" {
				ready = append(ready, issue)
			}
		}
		_ = json.NewEncoder(os.Stdout).Encode(ready)
		os.Exit(0)
	case "dep":
		if len(args) >= 4 && args[1] == "add" {
			_ = handleBdDepAdd(storePath, args[2], args[3])
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "usage: bd dep add <id> <dep>")
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "unknown bd command: %s\n", args[0])
		os.Exit(2)
	}
}

func runFakeTmux() {
	storePath := os.Getenv("MF_FAKE_TMUX_STORE")
	if storePath == "" {
		fmt.Fprintln(os.Stderr, "missing MF_FAKE_TMUX_STORE")
		os.Exit(2)
	}
	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(0)
	}
	cmd := args[0]
	store := loadTmuxStore(storePath)
	writeStore := func() {
		_ = saveTmuxStore(storePath, store)
	}
	findSession := func(flag string) string {
		for i := 0; i < len(args); i++ {
			if args[i] == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	switch cmd {
	case "has-session":
		session := findSession("-t")
		if session != "" && store[session] {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "no server running")
		os.Exit(1)
	case "new-session":
		session := findSession("-s")
		if session != "" {
			store[session] = true
			writeStore()
		}
		os.Exit(0)
	case "kill-session", "attach", "send-keys":
		session := findSession("-t")
		if session != "" && store[session] {
			if cmd == "kill-session" {
				delete(store, session)
				writeStore()
			}
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "can't find session")
		os.Exit(1)
	default:
		os.Exit(0)
	}
}

func handleBdCreate(storePath string, args []string) fakeIssue {
	store := loadStoreRaw(storePath)
	issue := fakeIssue{Status: "open"}
	deps := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
		case "-t":
			if i+1 < len(args) {
				issue.Type = args[i+1]
				i++
			}
		case "-p":
			if i+1 < len(args) {
				issue.Priority = args[i+1]
				i++
			}
		case "--status":
			if i+1 < len(args) {
				issue.Status = args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				issue.Description = args[i+1]
				i++
			}
		case "--deps":
			if i+1 < len(args) {
				deps = append(deps, args[i+1])
				i++
			}
		default:
		}
	}
	if len(args) > 0 {
		issue.Title = args[len(args)-1]
	}
	issue.ID = fmt.Sprintf("B%04d", store.NextID)
	store.NextID++
	issue.Deps = deps
	issue.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	store.Issues = append(store.Issues, issue)
	_ = saveStore(storePath, store)
	return issue
}

func handleBdUpdate(storePath string, args []string) fakeIssue {
	if len(args) < 1 {
		os.Exit(2)
	}
	id := args[0]
	status := ""
	description := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 < len(args) {
				status = args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		}
	}
	store := loadStoreRaw(storePath)
	for i := range store.Issues {
		if store.Issues[i].ID == id {
			if status != "" {
				store.Issues[i].Status = status
			}
			if description != "" {
				store.Issues[i].Description = description
			}
			_ = saveStore(storePath, store)
			return store.Issues[i]
		}
	}
	os.Exit(2)
	return fakeIssue{}
}

func handleBdClose(storePath string, args []string) fakeIssue {
	if len(args) < 1 {
		os.Exit(2)
	}
	id := args[0]
	store := loadStoreRaw(storePath)
	for i := range store.Issues {
		if store.Issues[i].ID == id {
			store.Issues[i].Status = "closed"
			_ = saveStore(storePath, store)
			return store.Issues[i]
		}
	}
	os.Exit(2)
	return fakeIssue{}
}

func handleBdShow(storePath string, args []string) fakeIssue {
	if len(args) < 1 {
		os.Exit(2)
	}
	id := args[0]
	store := loadStoreRaw(storePath)
	for i := range store.Issues {
		if store.Issues[i].ID == id {
			return store.Issues[i]
		}
	}
	os.Exit(2)
	return fakeIssue{}
}

func handleBdDepAdd(storePath, id, dep string) error {
	store := loadStoreRaw(storePath)
	for i := range store.Issues {
		if store.Issues[i].ID == id {
			store.Issues[i].Deps = append(store.Issues[i].Deps, dep)
			return saveStore(storePath, store)
		}
	}
	return nil
}

func saveStore(path string, store fakeStore) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func loadStoreRaw(path string) fakeStore {
	b, err := os.ReadFile(path)
	if err != nil {
		return fakeStore{NextID: 1}
	}
	var store fakeStore
	if err := json.Unmarshal(b, &store); err != nil {
		return fakeStore{NextID: 1}
	}
	if store.NextID == 0 {
		store.NextID = 1
	}
	return store
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func loadTmuxStore(path string) map[string]bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return map[string]bool{}
	}
	store := map[string]bool{}
	_ = json.Unmarshal(b, &store)
	return store
}

func saveTmuxStore(path string, store map[string]bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
