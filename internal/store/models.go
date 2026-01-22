package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type RigRow struct {
	ID, Name, RepoPath, TmuxPrefix, RuntimeCmd, CreatedAt string
	RuntimeArgs []string
}

type CellRow struct {
	ID, RigID, Name, ScopePrefix, WorktreePath, CreatedAt string
}

type AgentRow struct {
	ID, RigID, CellID, Role, Name, TmuxSession, Status, CreatedAt string
	LastSeenAt sql.NullString
}

type TaskRow struct {
	ID, RigID, Kind, Title, Body, Status, CreatedAt, UpdatedAt string
	ScopePrefix sql.NullString
}

type AssignmentRow struct {
	ID int64
	RigID, TaskID, AgentID, Status, InboxRel, OutboxRel, CompletionPromise, CreatedAt, UpdatedAt string
	RequestedBy sql.NullString
}

func randID(prefix string) string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b[:]))
}

func EnsureRig(db *sql.DB, cfg RigConfig) (RigRow, error) {
	var r RigRow
	var argsJSON string
	row := db.QueryRow("SELECT id, name, repo_path, tmux_prefix, runtime_cmd, runtime_args_json, created_at FROM rigs WHERE name = ?", cfg.Name)
	if err := row.Scan(&r.ID, &r.Name, &r.RepoPath, &r.TmuxPrefix, &r.RuntimeCmd, &argsJSON, &r.CreatedAt); err == nil {
		_ = json.Unmarshal([]byte(argsJSON), &r.RuntimeArgs)
		return r, nil
	}
	id := randID("rig")
	argsB, _ := json.Marshal(cfg.RuntimeArgs)
	_, err := db.Exec("INSERT INTO rigs(id,name,repo_path,tmux_prefix,runtime_cmd,runtime_args_json,created_at) VALUES(?,?,?,?,?,?,?)",
		id, cfg.Name, cfg.RepoPath, cfg.TmuxPrefix, cfg.RuntimeCmd, string(argsB), cfg.CreatedAt)
	if err != nil { return RigRow{}, err }
	return RigRow{ID: id, Name: cfg.Name, RepoPath: cfg.RepoPath, TmuxPrefix: cfg.TmuxPrefix, RuntimeCmd: cfg.RuntimeCmd, RuntimeArgs: cfg.RuntimeArgs, CreatedAt: cfg.CreatedAt}, nil
}

func GetRigByName(db *sql.DB, name string) (RigRow, error) {
	var r RigRow
	var argsJSON string
	row := db.QueryRow("SELECT id, name, repo_path, tmux_prefix, runtime_cmd, runtime_args_json, created_at FROM rigs WHERE name = ?", name)
	if err := row.Scan(&r.ID, &r.Name, &r.RepoPath, &r.TmuxPrefix, &r.RuntimeCmd, &argsJSON, &r.CreatedAt); err != nil {
		return RigRow{}, err
	}
	_ = json.Unmarshal([]byte(argsJSON), &r.RuntimeArgs)
	return r, nil
}

func CreateCell(db *sql.DB, rigID, name, scopePrefix, worktreePath string) (CellRow, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(scopePrefix) == "" {
		return CellRow{}, fmt.Errorf("cell name and scope are required")
	}
	id := randID("cell")
	created := MustNow()
	_, err := db.Exec("INSERT INTO cells(id,rig_id,name,scope_prefix,worktree_path,created_at) VALUES(?,?,?,?,?,?)",
		id, rigID, name, scopePrefix, worktreePath, created)
	if err != nil { return CellRow{}, err }
	return CellRow{ID: id, RigID: rigID, Name: name, ScopePrefix: scopePrefix, WorktreePath: worktreePath, CreatedAt: created}, nil
}

func GetCell(db *sql.DB, rigID, name string) (CellRow, error) {
	var c CellRow
	row := db.QueryRow("SELECT id, rig_id, name, scope_prefix, worktree_path, created_at FROM cells WHERE rig_id = ? AND name = ?", rigID, name)
	if err := row.Scan(&c.ID, &c.RigID, &c.Name, &c.ScopePrefix, &c.WorktreePath, &c.CreatedAt); err != nil { return CellRow{}, err }
	return c, nil
}

func CreateAgent(db *sql.DB, rigID, cellID, role, name, tmuxSession string) (AgentRow, error) {
	id := randID("agent")
	created := MustNow()
	_, err := db.Exec("INSERT INTO agents(id,rig_id,cell_id,role,name,tmux_session,status,created_at) VALUES(?,?,?,?,?,?,?,?)",
		id, rigID, cellID, role, name, tmuxSession, "idle", created)
	if err != nil { return AgentRow{}, err }
	return AgentRow{ID: id, RigID: rigID, CellID: cellID, Role: role, Name: name, TmuxSession: tmuxSession, Status: "idle", CreatedAt: created}, nil
}

func GetAgentByCellRole(db *sql.DB, cellID, role string) (AgentRow, error) {
	var a AgentRow
	row := db.QueryRow("SELECT id, rig_id, cell_id, role, name, tmux_session, status, last_seen_at, created_at FROM agents WHERE cell_id = ? AND role = ?", cellID, role)
	if err := row.Scan(&a.ID, &a.RigID, &a.CellID, &a.Role, &a.Name, &a.TmuxSession, &a.Status, &a.LastSeenAt, &a.CreatedAt); err != nil {
		return AgentRow{}, err
	}
	return a, nil
}

func CreateTask(db *sql.DB, rigID, kind, title, body, scope string) (TaskRow, error) {
	if strings.TrimSpace(kind) == "" { kind = "improve" }
	if strings.TrimSpace(title) == "" { return TaskRow{}, fmt.Errorf("title is required") }
	id := randID("task")
	now := MustNow()
	var scopeNS sql.NullString
	if strings.TrimSpace(scope) != "" { scopeNS = sql.NullString{String: scope, Valid: true} }
	_, err := db.Exec("INSERT INTO tasks(id,rig_id,kind,title,body,scope_prefix,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)",
		id, rigID, kind, title, body, scopeNS, "created", now, now)
	if err != nil { return TaskRow{}, err }
	return TaskRow{ID: id, RigID: rigID, Kind: kind, Title: title, Body: body, ScopePrefix: scopeNS, Status: "created", CreatedAt: now, UpdatedAt: now}, nil
}

func ListTasks(db *sql.DB, rigID string) ([]TaskRow, error) {
	rows, err := db.Query("SELECT id, rig_id, kind, title, body, scope_prefix, status, created_at, updated_at FROM tasks WHERE rig_id = ? ORDER BY created_at DESC", rigID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []TaskRow
	for rows.Next() {
		var t TaskRow
		if err := rows.Scan(&t.ID, &t.RigID, &t.Kind, &t.Title, &t.Body, &t.ScopePrefix, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func CreateAssignment(db *sql.DB, rigID, taskID, agentID, inboxRel, outboxRel, promise string, requestedBy *string) (AssignmentRow, error) {
	if strings.TrimSpace(promise) == "" { promise = "DONE" }
	now := MustNow()
	var req sql.NullString
	if requestedBy != nil && strings.TrimSpace(*requestedBy) != "" { req = sql.NullString{String: *requestedBy, Valid: true} }
	res, err := db.Exec(`INSERT INTO assignments(rig_id,task_id,agent_id,status,inbox_relpath,outbox_relpath,completion_promise,requested_by_agent_id,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`, rigID, taskID, agentID, "queued", inboxRel, outboxRel, promise, req, now, now)
	if err != nil { return AssignmentRow{}, err }
	id, _ := res.LastInsertId()
	return AssignmentRow{ID: id, RigID: rigID, TaskID: taskID, AgentID: agentID, Status: "queued", InboxRel: inboxRel, OutboxRel: outboxRel, CompletionPromise: promise, RequestedBy: req, CreatedAt: now, UpdatedAt: now}, nil
}

func ClaimNextAssignmentForAgent(db *sql.DB, agentID string) (AssignmentRow, bool, error) {
	row := db.QueryRow(`SELECT id, rig_id, task_id, agent_id, status, inbox_relpath, outbox_relpath, completion_promise, requested_by_agent_id, created_at, updated_at
FROM assignments WHERE agent_id = ? AND status = 'queued' ORDER BY id ASC LIMIT 1`, agentID)
	var a AssignmentRow
	if err := row.Scan(&a.ID, &a.RigID, &a.TaskID, &a.AgentID, &a.Status, &a.InboxRel, &a.OutboxRel, &a.CompletionPromise, &a.RequestedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if err == sql.ErrNoRows { return AssignmentRow{}, false, nil }
		return AssignmentRow{}, false, err
	}
	_, err := db.Exec("UPDATE assignments SET status='running', updated_at=? WHERE id=? AND status='queued'", MustNow(), a.ID)
	if err != nil { return AssignmentRow{}, false, err }
	a.Status = "running"
	return a, true, nil
}

func MarkAssignmentDone(db *sql.DB, assignmentID int64) error {
	_, err := db.Exec("UPDATE assignments SET status='done', updated_at=? WHERE id=?", MustNow(), assignmentID)
	return err
}

func MarkTaskDone(db *sql.DB, taskID string) error {
	_, err := db.Exec("UPDATE tasks SET status='done', updated_at=? WHERE id=?", MustNow(), taskID)
	return err
}

func MarkTaskAssigned(db *sql.DB, taskID string) error {
	_, err := db.Exec("UPDATE tasks SET status='assigned', updated_at=? WHERE id=?", MustNow(), taskID)
	return err
}
