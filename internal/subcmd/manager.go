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
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge manager tick <rig> [--watch]")
	}
	op := args[0]
	rest := args[1:]
	if op != "tick" {
		return fmt.Errorf("unknown manager subcommand: %s", op)
	}
	if len(rest) < 1 {
		return fmt.Errorf("usage: mforge manager tick <rig> [--watch]")
	}
	rigName := rest[0]
	watch := false
	for i := 1; i < len(rest); i++ {
		if rest[i] == "--watch" {
			watch = true
		}
	}
	for {
		updated, err := reconcile(home, rigName)
		if err != nil {
			return err
		}
		if updated > 0 {
			fmt.Printf("Reconciled %d assignment(s) to done\n", updated)
		}
		if !watch {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

func reconcile(home, rigName string) (int, error) {
	db, err := store.OpenDB(store.DBPath(home, rigName))
	if err != nil {
		return 0, err
	}
	defer db.Close()
	rigRow, err := store.GetRigByName(db, rigName)
	if err != nil {
		return 0, err
	}

	rows, err := db.Query(`
  SELECT a.id, a.task_id, a.agent_id, a.inbox_relpath, a.outbox_relpath, a.completion_promise, c.worktree_path
  FROM assignments a
  JOIN agents ag ON ag.id = a.agent_id
  JOIN cells c ON c.id = ag.cell_id
  WHERE a.rig_id = ? AND a.status IN ('running','queued')`, rigRow.ID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type pending struct {
		assnID   int64
		taskID   string
		inboxRel string
		outRel   string
		promise  string
		worktree string
	}
	var pendingList []pending
	for rows.Next() {
		var assnID int64
		var taskID, agentID, inboxRel, outRel, promise, worktree string
		if err := rows.Scan(&assnID, &taskID, &agentID, &inboxRel, &outRel, &promise, &worktree); err != nil {
			return 0, err
		}
		pendingList = append(pendingList, pending{
			assnID:   assnID,
			taskID:   taskID,
			inboxRel: inboxRel,
			outRel:   outRel,
			promise:  promise,
			worktree: worktree,
		})
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	updated := 0
	for _, item := range pendingList {
		outAbs := filepath.Join(item.worktree, item.outRel)
		b, err := os.ReadFile(outAbs)
		if err != nil {
			continue
		}
		if strings.Contains(string(b), item.promise) {
			_ = store.MarkAssignmentDone(db, item.assnID)
			_ = store.MarkTaskDone(db, item.taskID)
			archiveMail(item.worktree, item.inboxRel)
			archiveMail(item.worktree, item.outRel)
			updated++
		}
	}
	return updated, nil
}

func archiveMail(worktree, rel string) {
	if strings.TrimSpace(rel) == "" {
		return
	}
	src := filepath.Join(worktree, rel)
	dst := filepath.Join(worktree, "mail", "archive", filepath.Base(rel))
	_ = os.MkdirAll(filepath.Dir(dst), 0o755)
	_ = os.Rename(src, dst)
}
