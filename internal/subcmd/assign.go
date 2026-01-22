package subcmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/example/microforge/internal/store"
)

func Assign(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge assign <rig> --task <id> --cell <cell> --role <role> [--promise <token>]")
	}
	rigName := args[0]
	var taskID, cellName, role, promise string
	promise = "DONE"
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--task":
			if i+1 < len(args) {
				taskID = args[i+1]
				i++
			}
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
		case "--promise":
			if i+1 < len(args) {
				promise = args[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(taskID) == "" || strings.TrimSpace(cellName) == "" || strings.TrimSpace(role) == "" {
		return fmt.Errorf("--task, --cell, and --role are required")
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
	agentRow, err := store.GetAgentByCellRole(db, cellRow.ID, role)
	if err != nil {
		return err
	}

	inboxRel := filepath.Join("mail/inbox", fmt.Sprintf("%s.md", taskID))
	outboxRel := filepath.Join("mail/outbox", fmt.Sprintf("%s.md", taskID))
	if _, err := store.CreateAssignment(db, rigRow.ID, taskID, agentRow.ID, inboxRel, outboxRel, promise, nil); err != nil {
		return err
	}
	_ = store.MarkTaskAssigned(db, taskID)
	fmt.Printf("Assigned task %s -> %s/%s\n", taskID, cellName, role)
	return nil
}
