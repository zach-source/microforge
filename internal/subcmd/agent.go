package subcmd

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/hooks"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func Agent(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge agent <spawn|stop|attach|wake|relaunch|status|logs|heartbeat|create|bootstrap> ...")
	}
	op := args[0]
	rest := args[1:]
	if op == "create" {
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge agent create <path> --description <text> [--class crew|worker]")
		}
		return agentCreate(home, rest)
	}
	if op == "bootstrap" {
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge agent bootstrap <name>")
		}
		return agentBootstrap(home, rest)
	}
	if op == "status" {
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge agent status [--cell <cell>] [--role <role>] [--json]")
		}
		return agentStatus(home, rest)
	}
	if op == "logs" {
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge agent logs <cell> <role> [--follow] [--lines <n>] [--all]")
		}
		return agentLogs(home, rest)
	}
	if op == "heartbeat" {
		if len(rest) < 3 {
			return fmt.Errorf("usage: mforge agent heartbeat <cell> <role>")
		}
		return agentHeartbeat(home, rest)
	}
	if len(rest) < 3 {
		return fmt.Errorf("usage: mforge agent %s <cell> <role>", op)
	}
	rigName, cellName, role := rest[0], rest[1], rest[2]
	remote := false
	for i := 3; i < len(rest); i++ {
		if rest[i] == "--remote" {
			remote = true
		}
	}

	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
	if err != nil {
		return err
	}
	session := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, cellName, role)
	worktree := cellCfg.WorktreePath

	switch op {
	case "spawn":
		warnContextMismatch(home, rigName, "agent spawn")
		if err := ensureCellBootstrapped(home, rigName, cellName, role, false); err != nil {
			return err
		}
		if !remote && strings.TrimSpace(cfg.RemoteHost) == "" {
			if err := verifyWorktreeReady(worktree); err != nil {
				return err
			}
		}
		if err := setActiveAgent(worktree, role); err != nil {
			return err
		}
		if _, err := runTmux(cfg, remote, false, "has-session", "-t", session); err == nil {
			fmt.Printf("Session already running: %s\n", session)
			return nil
		}
		cmd, cmdArgs := runtimeForRole(cfg, role)
		cmdArgs = ensureSessionID(cfg, cmd, cmdArgs)
		remoteWorktree := resolveRemoteWorkdir(cfg, worktree, cellName)
		targs := []string{"new-session", "-d", "-s", session}
		if !remote && strings.TrimSpace(cfg.RemoteHost) == "" {
			targs = append(targs, awsEnvArgs()...)
		}
		targs = append(targs, "-c", remoteWorktree, "--", cmd)
		targs = append(targs, cmdArgs...)
		if _, err := runTmux(cfg, remote, false, targs...); err != nil {
			return err
		}
		_ = ensureAgentLogPipe(home, rigName, cellName, role, session, cfg, remote)
		writeHeartbeat(home, rigName, cellName, role, "spawned", "", "")
		maybeAcceptTrust(cfg, remote, session)
		emitOrchestrationEvent(cfg.RepoPath, beads.Meta{
			Cell:  cellName,
			Role:  role,
			Scope: cellCfg.ScopePrefix,
			Kind:  "agent_spawn",
		}, fmt.Sprintf("Agent spawned %s/%s", cellName, role), nil)
		fmt.Printf("Spawned %s\n", session)
		return nil

	case "stop":
		if _, err := runTmux(cfg, remote, false, "kill-session", "-t", session); err != nil {
			if isNoSessionErr(err) {
				return nil
			}
			return err
		}
		return nil

	case "attach":
		_, err := runTmux(cfg, remote, true, "attach", "-t", session)
		return err

	case "wake":
		warnContextMismatch(home, rigName, "agent wake")
		if err := ensureCellBootstrapped(home, rigName, cellName, role, false); err != nil {
			return err
		}
		if !remote && strings.TrimSpace(cfg.RemoteHost) == "" {
			if err := verifyWorktreeReady(worktree); err != nil {
				return err
			}
		}
		if err := setActiveAgent(worktree, role); err != nil {
			return err
		}
		if _, err := runTmux(cfg, remote, false, "has-session", "-t", session); err != nil {
			return fmt.Errorf("tmux session not running: %s", session)
		}
		_ = ensureAgentLogPipe(home, rigName, cellName, role, session, cfg, remote)
		writeHeartbeat(home, rigName, cellName, role, "woke", "", "")

		prompt := "Check mail/inbox/ for assignment .md files. Read the first one, work it, and write your report to the outbox file listed. If none, respond 'IDLE'."
		if _, err := runTmux(cfg, remote, false, "send-keys", "-t", session, prompt, "Enter"); err != nil {
			return err
		}
		emitOrchestrationEvent(cfg.RepoPath, beads.Meta{
			Cell:  cellName,
			Role:  role,
			Scope: cellCfg.ScopePrefix,
			Kind:  "agent_wake",
		}, fmt.Sprintf("Agent wake %s/%s", cellName, role), nil)
		fmt.Printf("Woke %s\n", session)
		return nil

	case "relaunch":
		warnContextMismatch(home, rigName, "agent relaunch")
		if err := ensureCellBootstrapped(home, rigName, cellName, role, false); err != nil {
			return err
		}
		if !remote && strings.TrimSpace(cfg.RemoteHost) == "" {
			if err := verifyWorktreeReady(worktree); err != nil {
				return err
			}
		}
		if err := setActiveAgent(worktree, role); err != nil {
			return err
		}
		if _, err := runTmux(cfg, remote, false, "kill-session", "-t", session); err != nil {
			if !isNoSessionErr(err) {
				return err
			}
		}
		cmd, cmdArgs := runtimeForRole(cfg, role)
		cmdArgs = ensureSessionID(cfg, cmd, cmdArgs)
		remoteWorktree := resolveRemoteWorkdir(cfg, worktree, cellName)
		targs := []string{"new-session", "-d", "-s", session}
		if !remote && strings.TrimSpace(cfg.RemoteHost) == "" {
			targs = append(targs, awsEnvArgs()...)
		}
		targs = append(targs, "-c", remoteWorktree, "--", cmd)
		targs = append(targs, cmdArgs...)
		if _, err := runTmux(cfg, remote, false, targs...); err != nil {
			return err
		}
		_ = ensureAgentLogPipe(home, rigName, cellName, role, session, cfg, remote)
		writeHeartbeat(home, rigName, cellName, role, "spawned", "", "")
		maybeAcceptTrust(cfg, remote, session)

		if _, err := runTmux(cfg, remote, false, "send-keys", "-t", session, "Check mail/inbox/ for assignment .md files. Read the first one, work it, and write your report to the outbox file listed. If none, respond 'IDLE'.", "Enter"); err != nil {
			return err
		}
		writeHeartbeat(home, rigName, cellName, role, "woke", "", "")
		emitOrchestrationEvent(cfg.RepoPath, beads.Meta{
			Cell:  cellName,
			Role:  role,
			Scope: cellCfg.ScopePrefix,
			Kind:  "agent_relaunch",
		}, fmt.Sprintf("Agent relaunch %s/%s", cellName, role), nil)
		fmt.Printf("Relaunched %s\n", session)
		return nil

	default:
		return fmt.Errorf("unknown agent subcommand: %s", op)
	}
}

func setActiveAgent(worktree, role string) error {
	idPath := filepath.Join(worktree, ".mf", "active-agent-"+role+".json")
	b, err := os.ReadFile(idPath)
	if err != nil {
		return err
	}
	return util.AtomicWriteFile(filepath.Join(worktree, ".mf", "active-agent.json"), b, 0o644)
}

func ensureCellBootstrapped(home, rigName, cellName, role string, auto bool) error {
	cellCfg, err := rig.LoadCellConfig(rig.CellConfigPath(home, rigName, cellName))
	if err != nil {
		return err
	}
	rolePath := filepath.Join(cellCfg.WorktreePath, ".mf", "active-agent-"+role+".json")
	settingsPath := rig.CellClaudeSettingsPath(home, rigName, cellName)
	if _, err := os.Stat(rolePath); err == nil {
		if _, err := os.Stat(settingsPath); err == nil {
			return nil
		}
	}
	if !auto {
		return fmt.Errorf("cell %q is not bootstrapped; run: mforge cell bootstrap %s %s", cellName, rigName, cellName)
	}
	args := []string{"bootstrap", rigName, cellName}
	switch strings.ToLower(role) {
	case "cell":
		args = append(args, "--single")
	case "architect":
		args = append(args, "--architect")
	}
	return Cell(home, args)
}

func awsEnvArgs() []string {
	keys := []string{
		"AWS_PROFILE", "AWS_DEFAULT_PROFILE",
		"AWS_REGION", "AWS_DEFAULT_REGION",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
		"AWS_SDK_LOAD_CONFIG",
	}
	out := []string{}
	for _, key := range keys {
		val := strings.TrimSpace(os.Getenv(key))
		if val == "" {
			continue
		}
		out = append(out, "-e", fmt.Sprintf("%s=%s", key, val))
	}
	return out
}

func verifyWorktreeReady(worktree string) error {
	if strings.TrimSpace(worktree) == "" {
		return fmt.Errorf("worktree path is empty")
	}
	if _, err := os.Stat(worktree); err != nil {
		return fmt.Errorf("worktree not found: %s", worktree)
	}
	if strings.TrimSpace(os.Getenv("MF_ALLOW_EMPTY_WORKTREE")) != "" ||
		strings.TrimSpace(os.Getenv("MF_TEST_MODE")) != "" ||
		strings.TrimSpace(os.Getenv("MF_FAKE_BD_STORE")) != "" {
		return worktreeReadWriteTest(worktree)
	}
	if !worktreeHasCode(worktree) {
		return fmt.Errorf("worktree has no codebase files: %s", worktree)
	}
	if err := worktreeReadWriteTest(worktree); err != nil {
		return err
	}
	return nil
}

func worktreeHasCode(worktree string) bool {
	if isGitWorktree(worktree) {
		if res, err := util.Run(nil, "git", "-C", worktree, "ls-files"); err == nil {
			if strings.TrimSpace(res.Stdout) != "" {
				return true
			}
		}
	}
	entries, err := os.ReadDir(worktree)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" || name == ".mf" || name == ".claude" || name == "mail" {
			continue
		}
		return true
	}
	return false
}

func isGitWorktree(worktree string) bool {
	_, err := util.Run(nil, "git", "-C", worktree, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func worktreeReadWriteTest(worktree string) error {
	tmpDir := filepath.Join(worktree, ".mf")
	if err := util.EnsureDir(tmpDir); err != nil {
		return err
	}
	path := filepath.Join(tmpDir, ".mforge_rw_check")
	if err := os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644); err != nil {
		return fmt.Errorf("worktree not writable: %s", worktree)
	}
	_ = os.Remove(path)
	return nil
}

func isNoSessionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "can't find session") || strings.Contains(msg, "no server running")
}

func agentStatus(home string, args []string) error {
	rigName := args[0]
	var cellName, role string
	remote := false
	jsonOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--cell":
			if i+1 < len(args) {
				cellName = args[i+1]
				i++
			}
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--remote":
			remote = true
		case "--json":
			jsonOut = true
		}
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	roles := []string{"builder", "monitor", "reviewer", "architect", "cell"}
	rows := make([]map[string]string, 0)
	for _, c := range cells {
		if cellName != "" && c.Name != cellName {
			continue
		}
		for _, r := range roles {
			if role != "" && r != role {
				continue
			}
			session := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, c.Name, r)
			state := "stopped"
			if _, err := runTmux(cfg, remote, false, "has-session", "-t", session); err == nil {
				state = "running"
			}
			hb := readHeartbeat(agentObsDir(home, rigName, c.Name, r))
			lastSeen := "-"
			status := "-"
			assignment := "-"
			lastLog := "-"
			if hb.Timestamp != "" {
				lastSeen = hb.Timestamp
			}
			if hb.Status != "" {
				status = hb.Status
			}
			if hb.AssignmentID != "" {
				assignment = hb.AssignmentID
			}
			logPath := filepath.Join(agentObsDir(home, rigName, c.Name, r), "agent.log")
			if lines, err := readLastLines(logPath, 1); err == nil && len(lines) == 1 {
				lastLog = sanitizeLogLine(lines[0])
			}
			if jsonOut {
				rows = append(rows, map[string]string{
					"cell":       c.Name,
					"role":       r,
					"session":    session,
					"state":      state,
					"last_seen":  lastSeen,
					"heartbeat":  status,
					"assignment": assignment,
					"last_log":   lastLog,
					"context":    "unknown",
				})
				continue
			}
			fmt.Printf("%s/%s\t%s\t%s\t%s\t%s\t%s\t%s\n", c.Name, r, session, state, lastSeen, status, assignment, lastLog)
		}
	}
	if jsonOut {
		b, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	}
	return nil
}

type AgentSpec struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	Class       string `json:"class,omitempty"`
	BeadID      string `json:"bead_id,omitempty"`
}

func agentCreate(home string, args []string) error {
	rigName := args[0]
	pathArg := args[1]
	var description string
	class := "worker"
	for i := 2; i < len(args); i++ {
		if args[i] == "--description" && i+1 < len(args) {
			description = args[i+1]
			i++
		}
		if args[i] == "--class" && i+1 < len(args) {
			class = strings.TrimSpace(args[i+1])
			i++
		}
	}
	if strings.TrimSpace(pathArg) == "" {
		return fmt.Errorf("agent path is required")
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	repoRoot := cfg.RepoPath
	absPath := pathArg
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(repoRoot, pathArg)
	}
	absPath = filepath.Clean(absPath)
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("agent path must be inside repo: %s", pathArg)
	}
	name := filepath.Base(absPath)
	scope := filepath.ToSlash(rel)
	if scope == "." {
		scope = ""
	}
	spec := AgentSpec{
		Name:        name,
		Path:        filepath.ToSlash(rel),
		Scope:       scope,
		Description: strings.TrimSpace(description),
		Class:       class,
	}
	if strings.TrimSpace(spec.Class) == "" {
		spec.Class = "worker"
	}
	agentID, err := ensureAgentBead(cfg.RepoPath, spec)
	if err != nil {
		return err
	}
	spec.BeadID = agentID
	specDir := filepath.Join(repoRoot, ".mf", "agents")
	if err := util.EnsureDir(specDir); err != nil {
		return err
	}
	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	if err := util.AtomicWriteFile(filepath.Join(specDir, name+".json"), b, 0o644); err != nil {
		return err
	}
	// ensure cell exists
	_ = Cell(home, []string{"add", rigName, name, "--scope", scope})
	fmt.Printf("Created agent %s (%s)\n", name, spec.Path)
	return nil
}

func agentBootstrap(home string, args []string) error {
	rigName := args[0]
	name := args[1]
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("agent name is required")
	}
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	repoRoot := cfg.RepoPath
	spec, err := loadAgentSpec(repoRoot, name)
	if err != nil {
		return err
	}
	_ = Cell(home, []string{"bootstrap", rigName, name, "--single"})
	absPath := filepath.Join(repoRoot, filepath.FromSlash(spec.Path))
	if err := writeAgentDocs(absPath, spec); err != nil {
		return err
	}
	fmt.Printf("Bootstrapped agent %s at %s\n", name, spec.Path)
	return nil
}

func loadAgentSpec(repoRoot, name string) (AgentSpec, error) {
	path := filepath.Join(repoRoot, ".mf", "agents", name+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return AgentSpec{}, err
	}
	var spec AgentSpec
	if err := json.Unmarshal(b, &spec); err != nil {
		return AgentSpec{}, err
	}
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = name
	}
	if strings.TrimSpace(spec.Path) == "" {
		spec.Path = name
	}
	if strings.TrimSpace(spec.Class) == "" {
		spec.Class = "worker"
	}
	return spec, nil
}

func writeAgentDocs(dir string, spec AgentSpec) error {
	if err := util.EnsureDir(dir); err != nil {
		return err
	}
	claudePath := filepath.Join(dir, "CLAUDE.md")
	agentPath := filepath.Join(dir, "AGENT.md")
	if _, err := os.Stat(claudePath); err != nil {
		content := fmt.Sprintf(`# Claude Instructions (%s)

Service: %s
Scope: %s

Focus on in-scope changes only. Communicate work via Beads (tasks/requests/observations).
`, spec.Name, spec.Name, spec.Scope)
		if err := util.AtomicWriteFile(claudePath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(agentPath); err != nil {
		content := fmt.Sprintf(`# Agent Guide (%s)

Description:
%s

Work model:
- This agent is a merged triad (build/review/monitor) unless split roles are used.
- Use Beads for assignments and status updates.
- Keep outputs concise and scoped.
`, spec.Name, strings.TrimSpace(spec.Description))
		if err := util.AtomicWriteFile(agentPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func agentLogs(home string, args []string) error {
	rigName := args[0]
	cellName := ""
	role := ""
	follow := false
	lines := 200
	all := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--follow":
			follow = true
		case "--lines":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil && v > 0 {
					lines = v
				}
				i++
			}
		case "--all":
			all = true
		default:
			if cellName == "" {
				cellName = args[i]
			} else if role == "" {
				role = args[i]
			}
		}
	}
	if all {
		if follow {
			return followAllLogs(home, rigName, lines)
		}
		linesByFile, err := readAllLogLines(home, rigName, lines)
		if err != nil {
			return err
		}
		for _, line := range linesByFile {
			fmt.Println(line)
		}
		return nil
	}
	if cellName == "" || role == "" {
		return fmt.Errorf("usage: mforge agent logs <cell> <role> [--follow] [--lines <n>] [--all]")
	}
	logPath := filepath.Join(agentObsDir(home, rigName, cellName, role), "agent.log")
	if follow {
		return tailFile(logPath, lines)
	}
	out, err := readLastLines(logPath, lines)
	if err != nil {
		return err
	}
	for _, line := range out {
		fmt.Println(line)
	}
	return nil
}

func agentHeartbeat(home string, args []string) error {
	rigName, cellName, role := args[0], args[1], args[2]
	path := filepath.Join(agentObsDir(home, rigName, cellName, role), "heartbeat.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func runtimeForRole(cfg rig.RigConfig, role string) (string, []string) {
	cmd := cfg.RuntimeCmd
	args := cfg.RuntimeArgs
	if spec, ok := cfg.RuntimeRoles[role]; ok {
		if strings.TrimSpace(spec.Cmd) != "" {
			cmd = spec.Cmd
		}
		if len(spec.Args) > 0 {
			args = spec.Args
		}
	}
	args = ensureDangerousSkip(cfg, cmd, args)
	return cmd, args
}

func ensureSessionID(cfg rig.RigConfig, cmd string, args []string) []string {
	if !strings.EqualFold(cfg.RuntimeProvider, "claude") && !strings.Contains(strings.ToLower(cmd), "claude") {
		return args
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "--session-id" && i+1 < len(args) {
			return args
		}
	}
	id := randomID(16)
	if strings.TrimSpace(id) == "" {
		return args
	}
	return append(args, "--session-id", id)
}

func ensureDangerousSkip(cfg rig.RigConfig, cmd string, args []string) []string {
	if !strings.EqualFold(cfg.RuntimeProvider, "claude") && !strings.Contains(strings.ToLower(cmd), "claude") {
		return args
	}
	for _, a := range args {
		if a == "--dangerously-skip-permissions" || a == "--dangerous-skip-permissions" {
			return args
		}
	}
	return append(args, "--dangerously-skip-permissions")
}

func randomID(n int) string {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return strings.ToLower(hex.EncodeToString(b))
}

func resolveRemoteWorkdir(cfg rig.RigConfig, localWorktree, cellName string) string {
	if strings.TrimSpace(cfg.RemoteWorkdir) == "" {
		return localWorktree
	}
	out := strings.ReplaceAll(cfg.RemoteWorkdir, "{cell}", cellName)
	return out
}

func runTmux(cfg rig.RigConfig, remote bool, tty bool, args ...string) (util.CmdResult, error) {
	if remote || strings.TrimSpace(cfg.RemoteHost) != "" {
		if strings.TrimSpace(cfg.RemoteHost) == "" {
			return util.CmdResult{}, fmt.Errorf("remote_host is not configured")
		}
		cmd, sshArgs := buildSSHCommand(cfg, append([]string{"tmux"}, args...), tty)
		return util.Run(nil, cmd, sshArgs...)
	}
	return util.Run(nil, "tmux", args...)
}

func agentObsDir(home, rigName, cellName, role string) string {
	return filepath.Join(home, "rigs", rigName, "agents", cellName, role)
}

func maybeAcceptTrust(cfg rig.RigConfig, remote bool, session string) {
	if remote || strings.TrimSpace(cfg.RemoteHost) != "" {
		return
	}
	time.Sleep(400 * time.Millisecond)
	_, _ = runTmux(cfg, false, false, "send-keys", "-t", session, "Enter")
}

func ensureAgentLogPipe(home, rigName, cellName, role, session string, cfg rig.RigConfig, remote bool) error {
	if remote || strings.TrimSpace(cfg.RemoteHost) != "" {
		return nil
	}
	dir := agentObsDir(home, rigName, cellName, role)
	if err := util.EnsureDir(dir); err != nil {
		return err
	}
	logPath := filepath.Join(dir, "agent.log")
	cmd := fmt.Sprintf("cat >> %s", logPath)
	_, err := runTmux(cfg, false, false, "pipe-pane", "-o", "-t", session, cmd)
	return err
}

func readHeartbeat(dir string) hooks.AgentHeartbeat {
	b, err := os.ReadFile(filepath.Join(dir, "heartbeat.json"))
	if err != nil {
		return hooks.AgentHeartbeat{}
	}
	var hb hooks.AgentHeartbeat
	if err := json.Unmarshal(b, &hb); err != nil {
		return hooks.AgentHeartbeat{}
	}
	return hb
}

func writeHeartbeat(home, rigName, cellName, role, status, assignmentID, message string) {
	dir := agentObsDir(home, rigName, cellName, role)
	_ = util.EnsureDir(dir)
	hb := hooks.AgentHeartbeat{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Status:       status,
		AssignmentID: assignmentID,
		Message:      message,
	}
	b, err := json.MarshalIndent(hb, "", "  ")
	if err != nil {
		return
	}
	_ = util.AtomicWriteFile(filepath.Join(dir, "heartbeat.json"), b, 0o644)
}

func readLastLines(path string, limit int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	lines := make([]string, 0, limit)
	for scanner.Scan() {
		line := scanner.Text()
		if len(lines) < limit {
			lines = append(lines, line)
			continue
		}
		copy(lines, lines[1:])
		lines[len(lines)-1] = line
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func tailFile(path string, lines int) error {
	args := []string{"-n", strconv.Itoa(lines), "-f", path}
	cmd := exec.Command("tail", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

type logTarget struct {
	Cell string
	Role string
	Path string
}

func listAgentLogTargets(home, rigName string) ([]logTarget, error) {
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return nil, err
	}
	roles := []string{"builder", "monitor", "reviewer", "architect", "cell"}
	out := make([]logTarget, 0)
	for _, cell := range cells {
		for _, role := range roles {
			path := filepath.Join(agentObsDir(home, rigName, cell.Name, role), "agent.log")
			out = append(out, logTarget{Cell: cell.Name, Role: role, Path: path})
		}
	}
	return out, nil
}

func readAllLogLines(home, rigName string, limit int) ([]string, error) {
	targets, err := listAgentLogTargets(home, rigName)
	if err != nil {
		return nil, err
	}
	lines := make([]string, 0)
	for _, target := range targets {
		out, err := readLastLines(target.Path, limit)
		if err != nil {
			continue
		}
		prefix := fmt.Sprintf("[%s/%s] ", target.Cell, target.Role)
		for _, line := range out {
			lines = append(lines, prefix+sanitizeLogLine(line))
		}
	}
	return lines, nil
}

func followAllLogs(home, rigName string, lines int) error {
	targets, err := listAgentLogTargets(home, rigName)
	if err != nil {
		return err
	}
	type cursor struct {
		offset  int64
		partial string
	}
	cursors := map[string]*cursor{}
	for _, target := range targets {
		out, err := readLastLines(target.Path, lines)
		if err == nil {
			prefix := fmt.Sprintf("[%s/%s] ", target.Cell, target.Role)
			for _, line := range out {
				fmt.Println(prefix + sanitizeLogLine(line))
			}
		}
		if info, err := os.Stat(target.Path); err == nil {
			cursors[target.Path] = &cursor{offset: info.Size()}
		} else {
			cursors[target.Path] = &cursor{offset: 0}
		}
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		for _, target := range targets {
			cur := cursors[target.Path]
			f, err := os.Open(target.Path)
			if err != nil {
				continue
			}
			if _, err := f.Seek(cur.offset, 0); err != nil {
				_ = f.Close()
				continue
			}
			buf, err := io.ReadAll(f)
			_ = f.Close()
			if err != nil || len(buf) == 0 {
				continue
			}
			cur.offset += int64(len(buf))
			content := cur.partial + string(buf)
			last := strings.LastIndex(content, "\n")
			chunk := content
			if last >= 0 {
				chunk = content[:last]
				cur.partial = content[last+1:]
			} else {
				cur.partial = content
				chunk = ""
			}
			lines := normalizeLogText(chunk)
			prefix := fmt.Sprintf("[%s/%s] ", target.Cell, target.Role)
			for _, line := range lines {
				fmt.Println(prefix + line)
			}
		}
	}
	return nil
}
