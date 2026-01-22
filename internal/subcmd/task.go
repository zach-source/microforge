package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/store"
)

func Task(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge task <create|list|split> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "create":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task create <rig> --title <t> [--body <md>] [--scope <path>] [--kind <kind>]")
		}
		rigName := rest[0]
		var title, body, scope, kind string
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
			case "--scope":
				if i+1 < len(rest) {
					scope = rest[i+1]
					i++
				}
			case "--kind":
				if i+1 < len(rest) {
					kind = rest[i+1]
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
		t, err := store.CreateTask(db, rigRow.ID, kind, title, body, scope)
		if err != nil {
			return err
		}
		fmt.Printf("Created task %s\n", t.ID)
		return nil

	case "list":
		if len(rest) != 1 {
			return fmt.Errorf("usage: mforge task list <rig>")
		}
		rigName := rest[0]
		db, err := store.OpenDB(store.DBPath(home, rigName))
		if err != nil {
			return err
		}
		defer db.Close()
		rigRow, err := store.GetRigByName(db, rigName)
		if err != nil {
			return err
		}
		tasks, err := store.ListTasks(db, rigRow.ID)
		if err != nil {
			return err
		}
		for _, t := range tasks {
			scope := ""
			if t.ScopePrefix.Valid {
				scope = t.ScopePrefix.String
			}
			fmt.Printf("%s\t%s\t%s\t%s", t.ID, t.Status, t.Kind, strings.ReplaceAll(t.Title, "\t", " "))
			if scope != "" {
				fmt.Printf("\t(scope=%s)", scope)
			}
			fmt.Println()
		}
		return nil

	case "split":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge task split <rig> --task <id> --cells <a,b,c>")
		}
		rigName := rest[0]
		var taskID, cellsCSV string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--task":
				if i+1 < len(rest) {
					taskID = rest[i+1]
					i++
				}
			case "--cells":
				if i+1 < len(rest) {
					cellsCSV = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(taskID) == "" || strings.TrimSpace(cellsCSV) == "" {
			return fmt.Errorf("--task and --cells are required")
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
		parent, err := store.GetTask(db, taskID)
		if err != nil {
			return err
		}
		cells := strings.Split(cellsCSV, ",")
		for _, c := range cells {
			cellName := strings.TrimSpace(c)
			if cellName == "" {
				continue
			}
			cellRow, err := store.GetCell(db, rigRow.ID, cellName)
			if err != nil {
				return err
			}
			title := fmt.Sprintf("Split: %s (%s)", parent.Title, cellName)
			body := fmt.Sprintf("Parent task: %s\n\n%s", parent.ID, parent.Body)
			scope := cellRow.ScopePrefix
			child, err := store.CreateTask(db, rigRow.ID, parent.Kind, title, body, scope)
			if err != nil {
				return err
			}
			if err := store.CreateTaskLink(db, parent.ID, child.ID); err != nil {
				return err
			}
			fmt.Printf("Created child task %s for %s\n", child.ID, cellName)
		}
		return nil

	default:
		return fmt.Errorf("unknown task subcommand: %s", op)
	}
}
