package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

type agentRow struct {
	Cell       string
	Role       string
	Session    string
	State      string
	LastSeen   string
	Heartbeat  string
	Assignment string
	Inbox      string
	LastLog    string
}

type roundStats struct {
	TasksOpen       int
	TasksInProgress int
	AssignmentsOpen int
	AssignmentsIP   int
	ReviewsOpen     int
}

type tuiModel struct {
	home     string
	rigName  string
	remote   bool
	interval time.Duration

	tab         int
	width       int
	height      int
	lastErr     string
	lastUpdated time.Time

	rows       []agentRow
	stats      roundStats
	turn       turn.State
	selected   int
	logLines   []string
	logErr     string
	logUpdated time.Time
	followAll  bool
	watchMode  bool
	watchRole  string
}

type dataMsg struct {
	rows  []agentRow
	stats roundStats
	turn  turn.State
	err   error
	when  time.Time
}

type tickMsg time.Time
type logMsg struct {
	lines []string
	err   error
	when  time.Time
}
type watchMsg struct {
	err error
}

func TUI(home string, args []string) error {
	interval := 2 * time.Second
	remote := false
	rigName := ""
	watchMode := false
	watchRole := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--interval":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil && v > 0 {
					interval = time.Duration(v) * time.Second
				}
				i++
			}
		case "--remote":
			remote = true
		case "--watch":
			watchMode = true
		case "--role":
			if i+1 < len(args) {
				watchRole = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && rigName == "" {
				rigName = args[i]
			}
		}
	}
	if strings.TrimSpace(rigName) == "" {
		return fmt.Errorf("usage: mforge tui [--interval <seconds>] [--remote]")
	}
	model := tuiModel{
		home:      home,
		rigName:   rigName,
		remote:    remote,
		interval:  interval,
		tab:       0,
		selected:  0,
		watchMode: watchMode,
		watchRole: watchRole,
	}
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m tuiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{loadDataCmd(m.home, m.rigName, m.remote), tickCmd(m.interval), loadLogsCmd(m.home, m.rigName, nil)}
	if m.watchMode {
		cmds = append(cmds, watchCmd(m.home, m.rigName, m.watchRole))
	}
	return tea.Batch(cmds...)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.tab = (m.tab + 1) % 4
			return m, loadLogsCmd(m.home, m.rigName, m)
		case "1":
			m.tab = 0
			return m, nil
		case "2":
			m.tab = 1
			return m, nil
		case "3":
			m.tab = 2
			return m, nil
		case "4":
			m.tab = 3
			return m, loadLogsCmd(m.home, m.rigName, m)
		case "j", "down":
			if m.tab == 3 && len(m.rows) > 0 {
				if m.selected < len(m.rows)-1 {
					m.selected++
				}
				return m, loadLogsCmd(m.home, m.rigName, m)
			}
		case "k", "up":
			if m.tab == 3 && len(m.rows) > 0 {
				if m.selected > 0 {
					m.selected--
				}
				return m, loadLogsCmd(m.home, m.rigName, m)
			}
		case "a":
			if m.tab == 3 {
				m.followAll = !m.followAll
				return m, loadLogsCmd(m.home, m.rigName, m)
			}
		case "r":
			return m, tea.Batch(loadDataCmd(m.home, m.rigName, m.remote), loadLogsCmd(m.home, m.rigName, m))
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		cmds := []tea.Cmd{loadDataCmd(m.home, m.rigName, m.remote), loadLogsCmd(m.home, m.rigName, m), tickCmd(m.interval)}
		if m.watchMode {
			cmds = append(cmds, watchCmd(m.home, m.rigName, m.watchRole))
		}
		return m, tea.Batch(cmds...)
	case dataMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
		} else {
			m.lastErr = ""
			m.rows = msg.rows
			m.stats = msg.stats
			m.turn = msg.turn
			m.lastUpdated = msg.when
			if m.selected >= len(m.rows) {
				m.selected = maxInt(0, len(m.rows)-1)
			}
		}
		return m, nil
	case logMsg:
		if msg.err != nil {
			m.logErr = msg.err.Error()
		} else {
			m.logErr = ""
			m.logLines = msg.lines
			m.logUpdated = msg.when
		}
		return m, nil
	case watchMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
		}
		return m, nil
	}
	return m, nil
}

func watchCmd(home, rigName, role string) tea.Cmd {
	return func() tea.Msg {
		err := watchOnce(home, rigName, role)
		return watchMsg{err: err}
	}
}

func (m tuiModel) View() string {
	header := lipgloss.NewStyle().Bold(true).Render("Microforge TUI")
	sub := fmt.Sprintf("Rig: %s  Updated: %s", m.rigName, m.lastUpdated.Format(time.RFC3339))
	if m.lastUpdated.IsZero() {
		sub = fmt.Sprintf("Rig: %s  Updated: -", m.rigName)
	}
	if m.lastErr != "" {
		sub += "  Error: " + m.lastErr
	}

	tabs := []string{"Agents", "Turn/Round", "Beads", "Logs"}
	tabRow := renderTabs(tabs, m.tab)

	body := ""
	switch m.tab {
	case 0:
		body = renderAgents(m.rows)
	case 1:
		body = renderTurnRound(m.turn, m.stats)
	case 2:
		body = renderBeadsSummary(m.stats)
	case 3:
		body = renderLogsWithMode(m.rows, m.selected, m.logLines, m.logErr, m.logUpdated, m.followAll)
	}

	footer := "Keys: [tab] switch  [1-4] go to tab  [j/k] select (Logs)  [a] toggle all  [r] refresh  [q] quit"

	return strings.Join([]string{
		header,
		sub,
		tabRow,
		"",
		body,
		"",
		footer,
	}, "\n")
}

func renderTabs(labels []string, active int) string {
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	out := make([]string, 0, len(labels))
	for i, label := range labels {
		if i == active {
			out = append(out, activeStyle.Render("["+label+"]"))
		} else {
			out = append(out, inactiveStyle.Render(" "+label+" "))
		}
	}
	return strings.Join(out, " ")
}

func renderAgents(rows []agentRow) string {
	if len(rows) == 0 {
		return "No agents found."
	}
	headers := []string{"cell", "role", "state", "session", "last_seen", "heartbeat", "assignment", "inbox", "last_log"}
	table := [][]string{headers}
	for _, row := range rows {
		table = append(table, []string{
			row.Cell,
			row.Role,
			row.State,
			trimSession(row.Session),
			row.LastSeen,
			row.Heartbeat,
			row.Assignment,
			row.Inbox,
			row.LastLog,
		})
	}
	return renderTable(table)
}

func renderTurnRound(state turn.State, stats roundStats) string {
	lines := []string{}
	if strings.TrimSpace(state.ID) == "" {
		lines = append(lines, "Turn: none")
	} else {
		lines = append(lines, fmt.Sprintf("Turn: %s  started=%s  status=%s", state.ID, state.StartedAt, strings.TrimSpace(state.Status)))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Tasks: open=%d  in_progress=%d", stats.TasksOpen, stats.TasksInProgress))
	lines = append(lines, fmt.Sprintf("Assignments: open=%d  in_progress=%d", stats.AssignmentsOpen, stats.AssignmentsIP))
	lines = append(lines, fmt.Sprintf("Reviews: open=%d", stats.ReviewsOpen))
	return strings.Join(lines, "\n")
}

func renderBeadsSummary(stats roundStats) string {
	return strings.Join([]string{
		"Open work:",
		fmt.Sprintf("- tasks open: %d", stats.TasksOpen),
		fmt.Sprintf("- tasks in_progress: %d", stats.TasksInProgress),
		fmt.Sprintf("- assignments open: %d", stats.AssignmentsOpen),
		fmt.Sprintf("- assignments in_progress: %d", stats.AssignmentsIP),
		fmt.Sprintf("- reviews open: %d", stats.ReviewsOpen),
	}, "\n")
}

func renderLogs(rows []agentRow, selected int, lines []string, logErr string, updated time.Time) string {
	return renderLogsWithMode(rows, selected, lines, logErr, updated, false)
}

func renderLogsWithMode(rows []agentRow, selected int, lines []string, logErr string, updated time.Time, all bool) string {
	if len(rows) == 0 {
		return "No agents available."
	}
	if selected < 0 || selected >= len(rows) {
		selected = 0
	}
	row := rows[selected]
	header := fmt.Sprintf("Selected: %s/%s  session=%s", row.Cell, row.Role, row.Session)
	if all {
		header = "Mode: follow all agents"
	}
	if !updated.IsZero() {
		header += "  updated=" + updated.Format(time.RFC3339)
	}
	if logErr != "" {
		header += "  error=" + logErr
	}
	body := "(no log data)"
	if len(lines) > 0 {
		body = strings.Join(lines, "\n")
	}
	return strings.Join([]string{header, "", body}, "\n")
}

func renderTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	colWidths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	lines := make([]string, 0, len(rows))
	for r, row := range rows {
		parts := make([]string, len(row))
		for i, cell := range row {
			parts[i] = padRight(cell, colWidths[i]+2)
		}
		line := strings.Join(parts, "")
		if r == 0 {
			line = lipgloss.NewStyle().Bold(true).Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func loadDataCmd(home, rigName string, remote bool) tea.Cmd {
	return func() tea.Msg {
		rows, stats, state, err := loadTUIData(home, rigName, remote)
		return dataMsg{
			rows:  rows,
			stats: stats,
			turn:  state,
			err:   err,
			when:  time.Now(),
		}
	}
}

func loadLogsCmd(home, rigName string, model any) tea.Cmd {
	m, ok := model.(tuiModel)
	if !ok {
		return func() tea.Msg {
			return logMsg{lines: nil, err: nil, when: time.Now()}
		}
	}
	if m.tab != 3 || len(m.rows) == 0 {
		return func() tea.Msg { return logMsg{lines: nil, err: nil, when: time.Now()} }
	}
	if m.followAll {
		return func() tea.Msg {
			lines, err := readAllAgentLogLines(home, rigName, m.rows, 200)
			return logMsg{lines: lines, err: err, when: time.Now()}
		}
	}
	if m.selected < 0 || m.selected >= len(m.rows) {
		return func() tea.Msg { return logMsg{lines: nil, err: nil, when: time.Now()} }
	}
	row := m.rows[m.selected]
	return func() tea.Msg {
		lines, err := readAgentLogLines(home, rigName, row.Cell, row.Role, 200)
		return logMsg{lines: lines, err: err, when: time.Now()}
	}
}

func loadTUIData(home, rigName string, remote bool) ([]agentRow, roundStats, turn.State, error) {
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return nil, roundStats{}, turn.State{}, err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return nil, roundStats{}, turn.State{}, err
	}
	cells, err := rig.ListCellConfigs(home, rigName)
	if err != nil {
		return nil, roundStats{}, turn.State{}, err
	}
	rows := collectAgentRows(home, rigName, cfg, cells, remote)
	stats := collectRoundStats(issues)
	state, _ := turn.Load(rig.TurnStatePath(home, rigName))
	return rows, stats, state, nil
}

func collectAgentRows(home, rigName string, cfg rig.RigConfig, cells []rig.CellConfig, remote bool) []agentRow {
	roles := []string{"builder", "monitor", "reviewer", "architect", "cell"}
	rows := make([]agentRow, 0)
	for _, cell := range cells {
		for _, role := range roles {
			session := fmt.Sprintf("%s-%s-%s-%s", cfg.TmuxPrefix, rigName, cell.Name, role)
			state := "stopped"
			if _, err := runTmux(cfg, remote, false, "has-session", "-t", session); err == nil {
				state = "running"
			}
			hb := readHeartbeat(agentObsDir(home, rigName, cell.Name, role))
			lastSeen := "-"
			status := "-"
			assignment := "-"
			if hb.Timestamp != "" {
				lastSeen = hb.Timestamp
			}
			if hb.Status != "" {
				status = hb.Status
			}
			if hb.AssignmentID != "" {
				assignment = hb.AssignmentID
			}
			inboxCount := countInbox(filepath.Join(cell.WorktreePath, "mail", "inbox"))
			lastLog := "-"
			logPath := filepath.Join(agentObsDir(home, rigName, cell.Name, role), "agent.log")
			if lines, err := readLastLines(logPath, 1); err == nil && len(lines) == 1 {
				lastLog = sanitizeLogLine(lines[0])
			}
			rows = append(rows, agentRow{
				Cell:       cell.Name,
				Role:       role,
				Session:    session,
				State:      state,
				LastSeen:   lastSeen,
				Heartbeat:  status,
				Assignment: assignment,
				Inbox:      fmt.Sprintf("%d", inboxCount),
				LastLog:    lastLog,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Cell == rows[j].Cell {
			return rows[i].Role < rows[j].Role
		}
		return rows[i].Cell < rows[j].Cell
	})
	return rows
}

func countInbox(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			count++
		}
	}
	return count
}

func collectRoundStats(issues []beads.Issue) roundStats {
	stats := roundStats{}
	for _, issue := range issues {
		switch strings.ToLower(issue.Type) {
		case "task":
			if issue.Status == "open" {
				stats.TasksOpen++
			}
			if issue.Status == "in_progress" {
				stats.TasksInProgress++
			}
		case "assignment":
			if issue.Status == "open" {
				stats.AssignmentsOpen++
			}
			if issue.Status == "in_progress" {
				stats.AssignmentsIP++
			}
		case "review":
			if issue.Status == "open" || issue.Status == "in_progress" {
				stats.ReviewsOpen++
			}
		}
	}
	return stats
}

func trimSession(session string) string {
	if len(session) <= 8 {
		return session
	}
	return session[len(session)-8:]
}

func readAgentLogLines(home, rigName, cellName, role string, limit int) ([]string, error) {
	path := filepath.Join(agentObsDir(home, rigName, cellName, role), "agent.log")
	lines, err := readLastLines(path, limit)
	if err != nil {
		return nil, err
	}
	return lines, nil
}

func readAllAgentLogLines(home, rigName string, rows []agentRow, limit int) ([]string, error) {
	lines := make([]string, 0)
	for _, row := range rows {
		out, err := readAgentLogLines(home, rigName, row.Cell, row.Role, limit)
		if err != nil {
			continue
		}
		prefix := fmt.Sprintf("[%s/%s] ", row.Cell, row.Role)
		for _, line := range out {
			lines = append(lines, prefix+line)
		}
	}
	return lines, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
