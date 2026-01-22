package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/store"
)

func Epic(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge epic <create|add-task|assign|status|close|conflict> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "create":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic create <rig> --title <t> [--body <md>]")
		}
		rigName := rest[0]
		var title, body string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--title":
				if i+1 < len(rest) {
					title = rest[i+1]
					i++
				}
			case "--body":
				if i+1 < len(rest) {
					body = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(title) == "" {
			return fmt.Errorf("--title is required")
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
		e, err := store.CreateEpic(db, rigRow.ID, title, body)
		if err != nil {
			return err
		}
		fmt.Printf("Created epic %s\n", e.ID)
		return nil

	case "add-task":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic add-task <rig> --epic <id> --task <id>")
		}
		rigName := rest[0]
		var epicID, taskID string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" || strings.TrimSpace(taskID) == "" {
			return fmt.Errorf("--epic and --task are required")
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
		ep, err := store.GetEpic(db, epicID)
		if err != nil {
			return err
		}
		if ep.RigID != rigRow.ID {
			return fmt.Errorf("epic not in rig %s", rigName)
		}
		if err := store.AddTaskToEpic(db, epicID, taskID); err != nil {
			return err
		}
		fmt.Printf("Added task %s to epic %s\n", taskID, epicID)
		return nil

	case "status":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic status <rig> --epic <id>")
		}
		rigName := rest[0]
		var epicID string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" {
			return fmt.Errorf("--epic is required")
		}
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil {
			return err
		}
		defer db.Close()
		rollup, err := store.EpicStatusRollup(db, epicID)
		if err != nil {
			return err
		}
		for status, count := range rollup {
			fmt.Printf("%s\t%d\n", status, count)
		}
		return nil

	case "assign":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic assign <rig> --epic <id> [--role <role>]")
		}
		rigName := rest[0]
		var epicID, role string
		role = "builder"
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			case "--role":
				if i+1 < len(rest) {
					role = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" {
			return fmt.Errorf("--epic is required")
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
		tasks, err := store.ListEpicTasks(db, epicID)
		if err != nil {
			return err
		}
		cells, err := store.ListCellsByRig(db, rigRow.ID)
		if err != nil {
			return err
		}
		assigned := 0
		for _, t := range tasks {
			if t.Status == "done" || t.Status == "assigned" {
				continue
			}
			scope := ""
			if t.ScopePrefix.Valid {
				scope = t.ScopePrefix.String
			}
			cell := matchCellByScope(cells, scope)
			if cell == nil {
				continue
			}
			agent, err := store.GetAgentByCellRole(db, cell.ID, role)
			if err != nil {
				return err
			}
			inboxRel := fmt.Sprintf("mail/inbox/%s.md", t.ID)
			outboxRel := fmt.Sprintf("mail/outbox/%s.md", t.ID)
			if _, err := store.CreateAssignment(db, rigRow.ID, t.ID, agent.ID, inboxRel, outboxRel, "DONE", nil); err != nil {
				return err
			}
			_ = store.MarkTaskAssigned(db, t.ID)
			assigned++
		}
		fmt.Printf("Assigned %d task(s)\n", assigned)
		return nil

	case "close":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic close <rig> --epic <id>")
		}
		rigName := rest[0]
		var epicID string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" {
			return fmt.Errorf("--epic is required")
		}
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil {
			return err
		}
		defer db.Close()
		tasks, err := store.ListEpicTasks(db, epicID)
		if err != nil {
			return err
		}
		hasReview := false
		for _, t := range tasks {
			if t.Status != "done" {
				return fmt.Errorf("epic has incomplete tasks")
			}
			if t.Kind == "review" {
				hasReview = true
			}
		}
		if !hasReview {
			return fmt.Errorf("epic close requires a completed review task")
		}
		if err := store.UpdateEpicStatus(db, epicID, "closed"); err != nil {
			return err
		}
		fmt.Printf("Closed epic %s\n", epicID)
		return nil

	case "conflict":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge epic conflict <rig> --epic <id> --cell <cell> --details <text>")
		}
		rigName := rest[0]
		var epicID, cellName, details string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--epic":
				if i+1 < len(rest) {
					epicID = rest[i+1]
					i++
				}
			case "--cell":
				if i+1 < len(rest) {
					cellName = rest[i+1]
					i++
				}
			case "--details":
				if i+1 < len(rest) {
					details = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(epicID) == "" || strings.TrimSpace(cellName) == "" {
			return fmt.Errorf("--epic and --cell are required")
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
		payload := fmt.Sprintf("{\"epic\":\"%s\",\"details\":%q}", epicID, details)
		if _, err := store.CreateRequest(db, rigRow.ID, cellRow.ID, "reviewer", "med", "p1", cellRow.ScopePrefix, payload); err != nil {
			return err
		}
		fmt.Printf("Created conflict request for epic %s\n", epicID)
		return nil

	default:
		return fmt.Errorf("unknown epic subcommand: %s", op)
	}
}

func matchCellByScope(cells []store.CellRow, scope string) *store.CellRow {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil
	}
	var best *store.CellRow
	for i := range cells {
		c := cells[i]
		if strings.HasPrefix(scope, c.ScopePrefix) {
			if best == nil || len(c.ScopePrefix) > len(best.ScopePrefix) {
				best = &c
			}
		}
	}
	return best
}
