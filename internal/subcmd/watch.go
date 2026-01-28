package subcmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/util"
)

func Watch(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge watch <rig> [--interval <seconds>] [--role <role>] [--fswatch] [--tui]")
	}
	rigName := args[0]
	interval := 60
	role := ""
	useFSWatch := false
	useTUI := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--interval":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil {
					interval = v
				}
				i++
			}
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--fswatch":
			useFSWatch = true
		case "--tui":
			useTUI = true
		}
	}
	if interval < 5 {
		interval = 5
	}
	if useTUI {
		tuiArgs := []string{rigName, "--interval", strconv.Itoa(interval), "--watch"}
		if strings.TrimSpace(role) != "" {
			tuiArgs = append(tuiArgs, "--role", role)
		}
		return TUI(home, tuiArgs)
	}
	if useFSWatch {
		return watchFS(home, rigName, role, time.Duration(interval)*time.Second)
	}
	for {
		if err := watchOnce(home, rigName, role); err != nil {
			return err
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func watchOnce(home, rigName, role string) error {
	warnContextMismatch(home, rigName, "watch")
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return err
	}
	roles := []string{"builder", "monitor", "reviewer", "architect", "cell"}
	if strings.TrimSpace(role) != "" {
		roles = []string{role}
	}
	for _, cell := range cells {
		for _, r := range roles {
			if !roleExists(cell.WorktreePath, r) {
				continue
			}
			session := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, cell.Name, r)
			_ = maybeAcceptTrustPrompt(home, rigName, cell.Name, r, session, cfg)
			inbox := filepath.Join(cell.WorktreePath, "mail", "inbox")
			pending := inboxCount(inbox)
			if pending == 0 {
				continue
			}
			if !shouldNudge(home, rigName, cell.Name, r) {
				continue
			}
			if _, err := runTmux(cfg, false, false, "has-session", "-t", session); err != nil {
				continue
			}
			prompt := fmt.Sprintf("New tasks detected (%d). Check mail/inbox and start the first task.", pending)
			_ = touchNudge(home, rigName, cell.Name, r)
			if err := sendWakePrompt(cfg, false, session, prompt); err != nil {
				return err
			}
		}
	}
	return nil
}

func watchFS(home, rigName, role string, interval time.Duration) error {
	paths, err := watchPaths(home, rigName)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no inbox paths found for rig %s", rigName)
	}
	if _, err := util.Run(nil, "fswatch", "--version"); err != nil {
		return fmt.Errorf("fswatch not available; install fswatch or run without --fswatch")
	}
	args := append([]string{"-0"}, paths...)
	cmd := exec.Command("fswatch", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()
	reader := bufio.NewReader(stdout)
	last := time.Time{}
	for {
		_, err := reader.ReadString(0)
		if err != nil {
			if err == io.EOF {
				return cmd.Wait()
			}
			return err
		}
		if time.Since(last) < interval {
			continue
		}
		last = time.Now()
		if err := watchOnce(home, rigName, role); err != nil {
			return err
		}
	}
}

func watchPaths(home, rigName string) ([]string, error) {
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(cells))
	for _, cell := range cells {
		path := filepath.Join(cell.WorktreePath, "mail", "inbox")
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func inboxCount(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			count++
		}
	}
	return count
}

func maybeAcceptTrustPrompt(home, rigName, cellName, role, session string, cfg rig.RigConfig) bool {
	if strings.TrimSpace(os.Getenv("MF_AUTO_TRUST")) == "0" {
		return false
	}
	if _, err := runTmux(cfg, false, false, "has-session", "-t", session); err != nil {
		return false
	}
	logPath := filepath.Join(agentObsDir(home, rigName, cellName, role), "agent.log")
	lines, err := readLastLinesLocal(logPath, 30)
	if err != nil {
		return false
	}
	if !trustPromptDetected(lines) {
		return false
	}
	if time.Since(readTrustNudge(home, rigName, cellName, role)) < 2*time.Minute {
		return false
	}
	_ = touchTrustNudge(home, rigName, cellName, role)
	_, _ = runTmux(cfg, false, false, "send-keys", "-t", session, "Enter")
	return true
}

func trustPromptDetected(lines []string) bool {
	for _, line := range lines {
		if strings.Contains(line, "Do you trust the files in this folder?") {
			return true
		}
	}
	return false
}

func shouldNudge(home, rigName, cellName, role string) bool {
	last := readNudge(home, rigName, cellName, role)
	if time.Since(last) < 5*time.Minute {
		return false
	}
	hb := readHeartbeat(agentObsDir(home, rigName, cellName, role))
	if hb.Timestamp == "" {
		return false
	}
	if strings.EqualFold(hb.Status, "claimed") || strings.EqualFold(hb.Status, "working") {
		return false
	}
	return true
}

func nudgePath(home, rigName, cellName, role string) string {
	return filepath.Join(home, "rigs", rigName, "agents", cellName, role, "last_nudge")
}

func readNudge(home, rigName, cellName, role string) time.Time {
	path := nudgePath(home, rigName, cellName, role)
	b, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	if err != nil {
		return time.Time{}
	}
	return ts
}

func touchNudge(home, rigName, cellName, role string) error {
	path := nudgePath(home, rigName, cellName, role)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
}

func trustNudgePath(home, rigName, cellName, role string) string {
	return filepath.Join(home, "rigs", rigName, "agents", cellName, role, "last_trust_nudge")
}

func readTrustNudge(home, rigName, cellName, role string) time.Time {
	path := trustNudgePath(home, rigName, cellName, role)
	b, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	if err != nil {
		return time.Time{}
	}
	return ts
}

func touchTrustNudge(home, rigName, cellName, role string) error {
	path := trustNudgePath(home, rigName, cellName, role)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
}

func readLastLinesLocal(path string, limit int) ([]string, error) {
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
