package subcmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/store"
)

type requestPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Kind  string `json:"kind"`
	Scope string `json:"scope"`
}

func Request(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge request <create|list|triage> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "create":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge request create <rig> --cell <cell> --role <role> --severity <sev> --priority <p> --scope <path> --payload <json>")
		}
		rigName := rest[0]
		var cellName, role, severity, priority, scope, payload string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--cell":
				if i+1 < len(rest) {
					cellName = rest[i+1]
					i++
				}
			case "--role":
				if i+1 < len(rest) {
					role = rest[i+1]
					i++
				}
			case "--severity":
				if i+1 < len(rest) {
					severity = rest[i+1]
					i++
				}
			case "--priority":
				if i+1 < len(rest) {
					priority = rest[i+1]
					i++
				}
			case "--scope":
				if i+1 < len(rest) {
					scope = rest[i+1]
					i++
				}
			case "--payload":
				if i+1 < len(rest) {
					payload = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(cellName) == "" || strings.TrimSpace(role) == "" {
			return fmt.Errorf("--cell and --role are required")
		}
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil {
			return err
		}
		defer db.Close()
		rigRow, err := store.GetRigByName(db, rigName)
		if err != nil {
			return err
		}
		cellRow, err := store.GetCell(db, rigRow.ID, cellName)
		if err != nil {
			return err
		}
		r, err := store.CreateRequest(db, rigRow.ID, cellRow.ID, role, severity, priority, scope, payload)
		if err != nil {
			return err
		}
		fmt.Printf("Created request %s\n", r.ID)
		return nil

	case "list":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge request list <rig> [--cell <cell>] [--status <status>] [--priority <p>]")
		}
		rigName := rest[0]
		var cellName, status, priority string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--cell":
				if i+1 < len(rest) {
					cellName = rest[i+1]
					i++
				}
			case "--status":
				if i+1 < len(rest) {
					status = rest[i+1]
					i++
				}
			case "--priority":
				if i+1 < len(rest) {
					priority = rest[i+1]
					i++
				}
			}
		}
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil {
			return err
		}
		defer db.Close()
		rigRow, err := store.GetRigByName(db, rigName)
		if err != nil {
			return err
		}
		var cellID *string
		if strings.TrimSpace(cellName) != "" {
			cellRow, err := store.GetCell(db, rigRow.ID, cellName)
			if err != nil {
				return err
			}
			cellID = &cellRow.ID
		}
		var statusPtr, priorityPtr *string
		if strings.TrimSpace(status) != "" {
			statusPtr = &status
		}
		if strings.TrimSpace(priority) != "" {
			priorityPtr = &priority
		}
		list, err := store.ListRequests(db, rigRow.ID, cellID, statusPtr, priorityPtr)
		if err != nil {
			return err
		}
		for _, r := range list {
			scope := ""
			if r.ScopePrefix.Valid {
				scope = r.ScopePrefix.String
			}
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s", r.ID, r.Status, r.Priority, r.Severity, r.SourceRole, r.CellID)
			if scope != "" {
				fmt.Printf("\t(scope=%s)", scope)
			}
			fmt.Println()
		}
		return nil

	case "triage":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge request triage <rig> --request <id> --action create-task|merge|block")
		}
		rigName := rest[0]
		var reqID, action string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--request":
				if i+1 < len(rest) {
					reqID = rest[i+1]
					i++
				}
			case "--action":
				if i+1 < len(rest) {
					action = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(reqID) == "" || strings.TrimSpace(action) == "" {
			return fmt.Errorf("--request and --action are required")
		}
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil {
			return err
		}
		defer db.Close()
		rigRow, err := store.GetRigByName(db, rigName)
		if err != nil {
			return err
		}
		req, err := store.GetRequest(db, reqID)
		if err != nil {
			return err
		}
		if req.RigID != rigRow.ID {
			return fmt.Errorf("request not in rig %s", rigName)
		}

		switch action {
		case "create-task":
			cellRow, err := store.GetCellByID(db, req.CellID)
			if err != nil {
				return err
			}
			agentRow, err := store.GetAgentByCellRole(db, cellRow.ID, "builder")
			if err != nil {
				return err
			}
			payload := requestPayload{}
			_ = json.Unmarshal([]byte(req.Payload), &payload)
			title := strings.TrimSpace(payload.Title)
			if title == "" {
				title = fmt.Sprintf("Request %s", req.ID)
			}
			body := strings.TrimSpace(payload.Body)
			if body == "" {
				body = req.Payload
			}
			kind := strings.TrimSpace(payload.Kind)
			scope := payload.Scope
			if scope == "" && req.ScopePrefix.Valid {
				scope = req.ScopePrefix.String
			}
			task, err := store.CreateTask(db, rigRow.ID, kind, title, body, scope)
			if err != nil {
				return err
			}
			inboxRel := fmt.Sprintf("mail/inbox/%s.md", task.ID)
			outboxRel := fmt.Sprintf("mail/outbox/%s.md", task.ID)
			if _, err := store.CreateAssignment(db, rigRow.ID, task.ID, agentRow.ID, inboxRel, outboxRel, "DONE", nil); err != nil {
				return err
			}
			_ = store.MarkTaskAssigned(db, task.ID)
			_ = store.UpdateRequestStatus(db, req.ID, "triaged")
			fmt.Printf("Triaged request %s -> task %s\n", req.ID, task.ID)
			return nil
		case "merge":
			return store.UpdateRequestStatus(db, req.ID, "merged")
		case "block":
			return store.UpdateRequestStatus(db, req.ID, "blocked")
		default:
			return fmt.Errorf("unknown action: %s", action)
		}

	default:
		return fmt.Errorf("unknown request subcommand: %s", op)
	}
}
