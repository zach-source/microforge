package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/microforge/internal/store"
)

func Manager(home string, args []string) error {
	if len(args) < 1 { return fmt.Errorf("usage: mf manager tick <rig> [--watch]") }
	op := args[0]
	rest := args[1:]
	if op != "tick" { return fmt.Errorf("unknown manager subcommand: %s", op) }
	if len(rest) < 1 { return fmt.Errorf("usage: mf manager tick <rig> [--watch]") }
	rigName := rest[0]
	watch := false
	for i := 1; i < len(rest); i++ {
		if rest[i] == "--watch" { watch = true }
	}
	for {
		updated, err := reconcile(home, rigName)
		if err != nil { return err }
		if updated > 0 { fmt.Printf("Reconciled %d assignment(s) to done\n", updated) }
		if !watch { return nil }
		time.Sleep(2 * time.Second)
	}
}

func reconcile(home, rigName string) (int, error) {
	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil { return 0, err }
	defer db.Close()
	rigRow, err := store.GetRigByName(db, rigName)
	if err != nil { return 0, err }

	rows, err := db.Query(`
  SELECT a.id, a.task_id, a.agent_id, a.outbox_relpath, a.completion_promise, c.worktree_path
  FROM assignments a
  JOIN agents ag ON ag.id = a.agent_id
  JOIN cells c ON c.id = ag.cell_id
  WHERE a.rig_id = ? AND a.status IN ('running','queued')`, rigRow.ID)
	if err != nil { return 0, err }
	defer rows.Close()
	updated := 0
	for rows.Next() {
		var assnID int64
		var taskID, agentID, outRel, promise, worktree string
		if err := rows.Scan(&assnID, &taskID, &agentID, &outRel, &promise, &worktree); err != nil { return updated, err }
		outAbs := filepath.Join(worktree, outRel)
		b, err := os.ReadFile(outAbs)
		if err != nil { continue }
		if strings.Contains(string(b), promise) {
			_ = store.MarkAssignmentDone(db, assnID)
			_ = store.MarkTaskDone(db, taskID)
			updated++
		}
	}
	return updated, nil
}
