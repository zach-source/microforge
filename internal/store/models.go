package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type RigRow struct {
	ID, Name, RepoPath, TmuxPrefix, RuntimeProvider, RuntimeCmd, CreatedAt string
	RuntimeArgs                                                            []string
	RuntimeRoles                                                           map[string]RuntimeSpec
	RemoteHost, RemoteUser, RemoteWorkdir, RemoteTmuxPrefix                string
	RemotePort                                                             int
	LibraryAddr, LibraryContext7URL, LibraryContext7Token                  string
	LibraryDocs                                                            []string
}

type CellRow struct {
	ID, RigID, Name, ScopePrefix, WorktreePath, CreatedAt string
}

type AgentRow struct {
	ID, RigID, CellID, Role, Name, TmuxSession, Status, CreatedAt string
	LastSeenAt                                                    sql.NullString
}

type TaskRow struct {
	ID, RigID, Kind, Title, Body, Status, CreatedAt, UpdatedAt string
	ScopePrefix                                                sql.NullString
}

type AssignmentRow struct {
	ID                                                                                           int64
	RigID, TaskID, AgentID, Status, InboxRel, OutboxRel, CompletionPromise, CreatedAt, UpdatedAt string
	RequestedBy                                                                                  sql.NullString
}

type RequestRow struct {
	ID, RigID, CellID, SourceRole, Severity, Priority, Status, Payload, CreatedAt, UpdatedAt string
	ScopePrefix                                                                              sql.NullString
}

type EpicRow struct {
	ID, RigID, Title, Body, Status, CreatedAt, UpdatedAt string
}

type TaskLinkRow struct {
	ParentTaskID, ChildTaskID, CreatedAt string
}

func randID(prefix string) string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b[:]))
}

func EnsureRig(db *sql.DB, cfg RigConfig) (RigRow, error) {
	var r RigRow
	var argsJSON, rolesJSON, docsJSON string
	row := db.QueryRow("SELECT id, name, repo_path, tmux_prefix, runtime_provider, runtime_cmd, runtime_args_json, runtime_roles_json, remote_host, remote_user, remote_port, remote_workdir, remote_tmux_prefix, library_addr, library_docs_json, library_context7_url, library_context7_token, created_at FROM rigs WHERE name = ?", cfg.Name)
	if err := row.Scan(&r.ID, &r.Name, &r.RepoPath, &r.TmuxPrefix, &r.RuntimeProvider, &r.RuntimeCmd, &argsJSON, &rolesJSON, &r.RemoteHost, &r.RemoteUser, &r.RemotePort, &r.RemoteWorkdir, &r.RemoteTmuxPrefix, &r.LibraryAddr, &docsJSON, &r.LibraryContext7URL, &r.LibraryContext7Token, &r.CreatedAt); err == nil {
		_ = json.Unmarshal([]byte(argsJSON), &r.RuntimeArgs)
		_ = json.Unmarshal([]byte(rolesJSON), &r.RuntimeRoles)
		_ = json.Unmarshal([]byte(docsJSON), &r.LibraryDocs)
		return r, nil
	}
	id := randID("rig")
	argsB, _ := json.Marshal(cfg.RuntimeArgs)
	rolesB, _ := json.Marshal(cfg.RuntimeRoles)
	docsB, _ := json.Marshal(cfg.LibraryDocs)
	_, err := db.Exec(`INSERT INTO rigs(
id,name,repo_path,tmux_prefix,runtime_provider,runtime_cmd,runtime_args_json,runtime_roles_json,
remote_host,remote_user,remote_port,remote_workdir,remote_tmux_prefix,
library_addr,library_docs_json,library_context7_url,library_context7_token,created_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, cfg.Name, cfg.RepoPath, cfg.TmuxPrefix, cfg.RuntimeProvider, cfg.RuntimeCmd, string(argsB), string(rolesB),
		cfg.RemoteHost, cfg.RemoteUser, cfg.RemotePort, cfg.RemoteWorkdir, cfg.RemoteTmuxPrefix,
		cfg.LibraryAddr, string(docsB), cfg.LibraryContext7URL, cfg.LibraryContext7Token, cfg.CreatedAt)
	if err != nil {
		return RigRow{}, err
	}
	return RigRow{ID: id, Name: cfg.Name, RepoPath: cfg.RepoPath, TmuxPrefix: cfg.TmuxPrefix, RuntimeProvider: cfg.RuntimeProvider, RuntimeCmd: cfg.RuntimeCmd, RuntimeArgs: cfg.RuntimeArgs, RuntimeRoles: cfg.RuntimeRoles, RemoteHost: cfg.RemoteHost, RemoteUser: cfg.RemoteUser, RemotePort: cfg.RemotePort, RemoteWorkdir: cfg.RemoteWorkdir, RemoteTmuxPrefix: cfg.RemoteTmuxPrefix, LibraryAddr: cfg.LibraryAddr, LibraryDocs: cfg.LibraryDocs, LibraryContext7URL: cfg.LibraryContext7URL, LibraryContext7Token: cfg.LibraryContext7Token, CreatedAt: cfg.CreatedAt}, nil
}

func GetRigByName(db *sql.DB, name string) (RigRow, error) {
	var r RigRow
	var argsJSON, rolesJSON, docsJSON string
	row := db.QueryRow("SELECT id, name, repo_path, tmux_prefix, runtime_provider, runtime_cmd, runtime_args_json, runtime_roles_json, remote_host, remote_user, remote_port, remote_workdir, remote_tmux_prefix, library_addr, library_docs_json, library_context7_url, library_context7_token, created_at FROM rigs WHERE name = ?", name)
	if err := row.Scan(&r.ID, &r.Name, &r.RepoPath, &r.TmuxPrefix, &r.RuntimeProvider, &r.RuntimeCmd, &argsJSON, &rolesJSON, &r.RemoteHost, &r.RemoteUser, &r.RemotePort, &r.RemoteWorkdir, &r.RemoteTmuxPrefix, &r.LibraryAddr, &docsJSON, &r.LibraryContext7URL, &r.LibraryContext7Token, &r.CreatedAt); err != nil {
		return RigRow{}, err
	}
	_ = json.Unmarshal([]byte(argsJSON), &r.RuntimeArgs)
	_ = json.Unmarshal([]byte(rolesJSON), &r.RuntimeRoles)
	_ = json.Unmarshal([]byte(docsJSON), &r.LibraryDocs)
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
	if err != nil {
		return CellRow{}, err
	}
	return CellRow{ID: id, RigID: rigID, Name: name, ScopePrefix: scopePrefix, WorktreePath: worktreePath, CreatedAt: created}, nil
}

func GetCell(db *sql.DB, rigID, name string) (CellRow, error) {
	var c CellRow
	row := db.QueryRow("SELECT id, rig_id, name, scope_prefix, worktree_path, created_at FROM cells WHERE rig_id = ? AND name = ?", rigID, name)
	if err := row.Scan(&c.ID, &c.RigID, &c.Name, &c.ScopePrefix, &c.WorktreePath, &c.CreatedAt); err != nil {
		return CellRow{}, err
	}
	return c, nil
}

func GetCellByID(db *sql.DB, id string) (CellRow, error) {
	var c CellRow
	row := db.QueryRow("SELECT id, rig_id, name, scope_prefix, worktree_path, created_at FROM cells WHERE id = ?", id)
	if err := row.Scan(&c.ID, &c.RigID, &c.Name, &c.ScopePrefix, &c.WorktreePath, &c.CreatedAt); err != nil {
		return CellRow{}, err
	}
	return c, nil
}

func ListCellsByRig(db *sql.DB, rigID string) ([]CellRow, error) {
	rows, err := db.Query("SELECT id, rig_id, name, scope_prefix, worktree_path, created_at FROM cells WHERE rig_id = ? ORDER BY created_at DESC", rigID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CellRow
	for rows.Next() {
		var c CellRow
		if err := rows.Scan(&c.ID, &c.RigID, &c.Name, &c.ScopePrefix, &c.WorktreePath, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func CreateAgent(db *sql.DB, rigID, cellID, role, name, tmuxSession string) (AgentRow, error) {
	id := randID("agent")
	created := MustNow()
	_, err := db.Exec("INSERT INTO agents(id,rig_id,cell_id,role,name,tmux_session,status,created_at) VALUES(?,?,?,?,?,?,?,?)",
		id, rigID, cellID, role, name, tmuxSession, "idle", created)
	if err != nil {
		return AgentRow{}, err
	}
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

func ListAgentsByRig(db *sql.DB, rigID string) ([]AgentRow, error) {
	rows, err := db.Query("SELECT id, rig_id, cell_id, role, name, tmux_session, status, last_seen_at, created_at FROM agents WHERE rig_id = ? ORDER BY created_at DESC", rigID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AgentRow
	for rows.Next() {
		var a AgentRow
		if err := rows.Scan(&a.ID, &a.RigID, &a.CellID, &a.Role, &a.Name, &a.TmuxSession, &a.Status, &a.LastSeenAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func UpdateAgentLastSeen(db *sql.DB, agentID string) error {
	_, err := db.Exec("UPDATE agents SET last_seen_at=? WHERE id=?", MustNow(), agentID)
	return err
}

func CreateTask(db *sql.DB, rigID, kind, title, body, scope string) (TaskRow, error) {
	if strings.TrimSpace(kind) == "" {
		kind = "improve"
	}
	if strings.TrimSpace(title) == "" {
		return TaskRow{}, fmt.Errorf("title is required")
	}
	id := randID("task")
	now := MustNow()
	var scopeNS sql.NullString
	if strings.TrimSpace(scope) != "" {
		scopeNS = sql.NullString{String: scope, Valid: true}
	}
	_, err := db.Exec("INSERT INTO tasks(id,rig_id,kind,title,body,scope_prefix,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)",
		id, rigID, kind, title, body, scopeNS, "created", now, now)
	if err != nil {
		return TaskRow{}, err
	}
	return TaskRow{ID: id, RigID: rigID, Kind: kind, Title: title, Body: body, ScopePrefix: scopeNS, Status: "created", CreatedAt: now, UpdatedAt: now}, nil
}

func ListTasks(db *sql.DB, rigID string) ([]TaskRow, error) {
	rows, err := db.Query("SELECT id, rig_id, kind, title, body, scope_prefix, status, created_at, updated_at FROM tasks WHERE rig_id = ? ORDER BY created_at DESC", rigID)
	if err != nil {
		return nil, err
	}
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

func GetTask(db *sql.DB, id string) (TaskRow, error) {
	var t TaskRow
	row := db.QueryRow("SELECT id, rig_id, kind, title, body, scope_prefix, status, created_at, updated_at FROM tasks WHERE id = ?", id)
	if err := row.Scan(&t.ID, &t.RigID, &t.Kind, &t.Title, &t.Body, &t.ScopePrefix, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return TaskRow{}, err
	}
	return t, nil
}

func CreateAssignment(db *sql.DB, rigID, taskID, agentID, inboxRel, outboxRel, promise string, requestedBy *string) (AssignmentRow, error) {
	if strings.TrimSpace(promise) == "" {
		promise = "DONE"
	}
	now := MustNow()
	var req sql.NullString
	if requestedBy != nil && strings.TrimSpace(*requestedBy) != "" {
		req = sql.NullString{String: *requestedBy, Valid: true}
	}
	res, err := db.Exec(`INSERT INTO assignments(rig_id,task_id,agent_id,status,inbox_relpath,outbox_relpath,completion_promise,requested_by_agent_id,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`, rigID, taskID, agentID, "queued", inboxRel, outboxRel, promise, req, now, now)
	if err != nil {
		return AssignmentRow{}, err
	}
	id, _ := res.LastInsertId()
	return AssignmentRow{ID: id, RigID: rigID, TaskID: taskID, AgentID: agentID, Status: "queued", InboxRel: inboxRel, OutboxRel: outboxRel, CompletionPromise: promise, RequestedBy: req, CreatedAt: now, UpdatedAt: now}, nil
}

func ClaimNextAssignmentForAgent(db *sql.DB, agentID string) (AssignmentRow, bool, error) {
	var out AssignmentRow
	var claimed bool
	err := WithTx(context.Background(), db, func(tx *sql.Tx) error {
		row := tx.QueryRow(`SELECT id, rig_id, task_id, agent_id, status, inbox_relpath, outbox_relpath, completion_promise, requested_by_agent_id, created_at, updated_at
FROM assignments WHERE agent_id = ? AND status = 'queued' ORDER BY id ASC LIMIT 1`, agentID)
		if err := row.Scan(&out.ID, &out.RigID, &out.TaskID, &out.AgentID, &out.Status, &out.InboxRel, &out.OutboxRel, &out.CompletionPromise, &out.RequestedBy, &out.CreatedAt, &out.UpdatedAt); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		res, err := tx.Exec("UPDATE assignments SET status='running', updated_at=? WHERE id=? AND status='queued'", MustNow(), out.ID)
		if err != nil {
			return err
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return nil
		}
		out.Status = "running"
		claimed = true
		return nil
	})
	if err != nil {
		return AssignmentRow{}, false, err
	}
	if !claimed {
		return AssignmentRow{}, false, nil
	}
	return out, true, nil
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

func CreateRequest(db *sql.DB, rigID, cellID, sourceRole, severity, priority string, scope string, payload string) (RequestRow, error) {
	if strings.TrimSpace(sourceRole) == "" {
		return RequestRow{}, fmt.Errorf("source role is required")
	}
	if strings.TrimSpace(severity) == "" {
		severity = "med"
	}
	if strings.TrimSpace(priority) == "" {
		priority = "p2"
	}
	if strings.TrimSpace(payload) == "" {
		payload = "{}"
	}
	id := randID("req")
	now := MustNow()
	var scopeNS sql.NullString
	if strings.TrimSpace(scope) != "" {
		scopeNS = sql.NullString{String: scope, Valid: true}
	}
	_, err := db.Exec(`INSERT INTO requests(id,rig_id,cell_id,source_role,severity,priority,scope_prefix,payload_json,status,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, rigID, cellID, sourceRole, severity, priority, scopeNS, payload, "new", now, now)
	if err != nil {
		return RequestRow{}, err
	}
	return RequestRow{ID: id, RigID: rigID, CellID: cellID, SourceRole: sourceRole, Severity: severity, Priority: priority, ScopePrefix: scopeNS, Payload: payload, Status: "new", CreatedAt: now, UpdatedAt: now}, nil
}

func GetRequest(db *sql.DB, id string) (RequestRow, error) {
	var r RequestRow
	row := db.QueryRow("SELECT id, rig_id, cell_id, source_role, severity, priority, scope_prefix, payload_json, status, created_at, updated_at FROM requests WHERE id = ?", id)
	if err := row.Scan(&r.ID, &r.RigID, &r.CellID, &r.SourceRole, &r.Severity, &r.Priority, &r.ScopePrefix, &r.Payload, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return RequestRow{}, err
	}
	return r, nil
}

func ListRequests(db *sql.DB, rigID string, cellID *string, status *string, priority *string) ([]RequestRow, error) {
	q := "SELECT id, rig_id, cell_id, source_role, severity, priority, scope_prefix, payload_json, status, created_at, updated_at FROM requests WHERE rig_id = ?"
	args := []any{rigID}
	if cellID != nil && strings.TrimSpace(*cellID) != "" {
		q += " AND cell_id = ?"
		args = append(args, *cellID)
	}
	if status != nil && strings.TrimSpace(*status) != "" {
		q += " AND status = ?"
		args = append(args, *status)
	}
	if priority != nil && strings.TrimSpace(*priority) != "" {
		q += " AND priority = ?"
		args = append(args, *priority)
	}
	q += " ORDER BY created_at DESC"
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RequestRow
	for rows.Next() {
		var r RequestRow
		if err := rows.Scan(&r.ID, &r.RigID, &r.CellID, &r.SourceRole, &r.Severity, &r.Priority, &r.ScopePrefix, &r.Payload, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func UpdateRequestStatus(db *sql.DB, id, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("status is required")
	}
	_, err := db.Exec("UPDATE requests SET status=?, updated_at=? WHERE id=?", status, MustNow(), id)
	return err
}

func CreateEpic(db *sql.DB, rigID, title, body string) (EpicRow, error) {
	if strings.TrimSpace(title) == "" {
		return EpicRow{}, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(body) == "" {
		body = "(no description)"
	}
	id := randID("epic")
	now := MustNow()
	_, err := db.Exec("INSERT INTO epics(id,rig_id,title,body,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)",
		id, rigID, title, body, "open", now, now)
	if err != nil {
		return EpicRow{}, err
	}
	return EpicRow{ID: id, RigID: rigID, Title: title, Body: body, Status: "open", CreatedAt: now, UpdatedAt: now}, nil
}

func GetEpic(db *sql.DB, id string) (EpicRow, error) {
	var e EpicRow
	row := db.QueryRow("SELECT id, rig_id, title, body, status, created_at, updated_at FROM epics WHERE id = ?", id)
	if err := row.Scan(&e.ID, &e.RigID, &e.Title, &e.Body, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return EpicRow{}, err
	}
	return e, nil
}

func AddTaskToEpic(db *sql.DB, epicID, taskID string) error {
	_, err := db.Exec("INSERT OR IGNORE INTO epic_tasks(epic_id, task_id, created_at) VALUES(?,?,?)", epicID, taskID, MustNow())
	return err
}

func ListEpicTasks(db *sql.DB, epicID string) ([]TaskRow, error) {
	rows, err := db.Query(`SELECT t.id, t.rig_id, t.kind, t.title, t.body, t.scope_prefix, t.status, t.created_at, t.updated_at
FROM tasks t
JOIN epic_tasks et ON et.task_id = t.id
WHERE et.epic_id = ?
ORDER BY t.created_at DESC`, epicID)
	if err != nil {
		return nil, err
	}
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

func EpicStatusRollup(db *sql.DB, epicID string) (map[string]int, error) {
	rows, err := db.Query(`SELECT t.status, COUNT(1) FROM tasks t
JOIN epic_tasks et ON et.task_id = t.id
WHERE et.epic_id = ?
GROUP BY t.status`, epicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, nil
}

func CreateTaskLink(db *sql.DB, parentTaskID, childTaskID string) error {
	if strings.TrimSpace(parentTaskID) == "" || strings.TrimSpace(childTaskID) == "" {
		return fmt.Errorf("parent and child task ids are required")
	}
	_, err := db.Exec("INSERT OR IGNORE INTO task_links(parent_task_id, child_task_id, created_at) VALUES(?,?,?)", parentTaskID, childTaskID, MustNow())
	return err
}

func UpdateEpicStatus(db *sql.DB, epicID, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("status is required")
	}
	_, err := db.Exec("UPDATE epics SET status=?, updated_at=? WHERE id=?", status, MustNow(), epicID)
	return err
}

func CountRequestsByStatus(db *sql.DB, rigID string, cellID *string) (map[string]int, error) {
	q := "SELECT status, COUNT(1) FROM requests WHERE rig_id = ?"
	args := []any{rigID}
	if cellID != nil && strings.TrimSpace(*cellID) != "" {
		q += " AND cell_id = ?"
		args = append(args, *cellID)
	}
	q += " GROUP BY status"
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, nil
}

func CountTasksByStatus(db *sql.DB, rigID string, scopePrefix string) (map[string]int, error) {
	q := "SELECT status, COUNT(1) FROM tasks WHERE rig_id = ?"
	args := []any{rigID}
	if strings.TrimSpace(scopePrefix) != "" {
		q += " AND scope_prefix LIKE ?"
		args = append(args, scopePrefix+"%")
	}
	q += " GROUP BY status"
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, nil
}

func OldestRequestCreatedAt(db *sql.DB, rigID string, cellID *string) (sql.NullString, error) {
	q := "SELECT MIN(created_at) FROM requests WHERE rig_id = ?"
	args := []any{rigID}
	if cellID != nil && strings.TrimSpace(*cellID) != "" {
		q += " AND cell_id = ?"
		args = append(args, *cellID)
	}
	row := db.QueryRow(q, args...)
	var out sql.NullString
	if err := row.Scan(&out); err != nil {
		return sql.NullString{}, err
	}
	return out, nil
}

func OldestTaskCreatedAt(db *sql.DB, rigID string, scopePrefix string) (sql.NullString, error) {
	q := "SELECT MIN(created_at) FROM tasks WHERE rig_id = ?"
	args := []any{rigID}
	if strings.TrimSpace(scopePrefix) != "" {
		q += " AND scope_prefix LIKE ?"
		args = append(args, scopePrefix+"%")
	}
	row := db.QueryRow(q, args...)
	var out sql.NullString
	if err := row.Scan(&out); err != nil {
		return sql.NullString{}, err
	}
	return out, nil
}
