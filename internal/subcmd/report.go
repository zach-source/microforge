package subcmd

import (
	"fmt"
	"strings"

	"github.com/example/microforge/internal/store"
)

func Report(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge report <rig> [--cell <cell>]")
	}
	rigName := args[0]
	var cellName string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--cell":
			if i+1 < len(args) {
				cellName = args[i+1]
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
	scopePrefix := ""
	if strings.TrimSpace(cellName) != "" {
		cellRow, err := store.GetCell(db, rigRow.ID, cellName)
		if err != nil {
			return err
		}
		cellID = &cellRow.ID
		scopePrefix = cellRow.ScopePrefix
	}

	reqCounts, err := store.CountRequestsByStatus(db, rigRow.ID, cellID)
	if err != nil {
		return err
	}
	taskCounts, err := store.CountTasksByStatus(db, rigRow.ID, scopePrefix)
	if err != nil {
		return err
	}
	oldReq, err := store.OldestRequestCreatedAt(db, rigRow.ID, cellID)
	if err != nil {
		return err
	}
	oldTask, err := store.OldestTaskCreatedAt(db, rigRow.ID, scopePrefix)
	if err != nil {
		return err
	}

	fmt.Println("Requests")
	for status, count := range reqCounts {
		fmt.Printf("%s\t%d\n", status, count)
	}
	if oldReq.Valid {
		fmt.Printf("oldest_request\t%s\n", oldReq.String)
	}
	fmt.Println("Tasks")
	for status, count := range taskCounts {
		fmt.Printf("%s\t%d\n", status, count)
	}
	if oldTask.Valid {
		fmt.Printf("oldest_task\t%s\n", oldTask.String)
	}
	return nil
}
