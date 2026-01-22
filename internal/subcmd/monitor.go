package subcmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/store"
	"github.com/example/microforge/internal/util"
)

func Monitor(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge monitor <run-tests> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "run-tests":
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge monitor run-tests <rig> <cell> --cmd <command...> [--severity <sev>] [--priority <p>] [--scope <path>]")
		}
		rigName, cellName := rest[0], rest[1]
		var cmdParts []string
		severity := "high"
		priority := "p1"
		scope := ""
		for i := 2; i < len(rest); i++ {
			switch rest[i] {
			case "--cmd":
				for j := i + 1; j < len(rest); j++ {
					cmdParts = append(cmdParts, rest[j])
				}
				i = len(rest)
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
			}
		}
		if len(cmdParts) == 0 {
			return fmt.Errorf("--cmd is required")
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

		cmd := cmdParts[0]
		cmdArgs := cmdParts[1:]
		res, err := util.RunInDir(nil, cellRow.WorktreePath, cmd, cmdArgs...)
		if err == nil {
			fmt.Println("OK")
			return nil
		}

		payload := map[string]any{
			"title":   "Monitor failure: " + strings.Join(cmdParts, " "),
			"body":    "Command failed: " + strings.Join(cmdParts, " "),
			"kind":    "monitor",
			"scope":   scope,
			"command": strings.Join(cmdParts, " "),
			"stdout":  res.Stdout,
			"stderr":  res.Stderr,
		}
		if strings.TrimSpace(scope) == "" {
			payload["scope"] = cellRow.ScopePrefix
		}
		payloadJSON, _ := json.Marshal(payload)
		_, reqErr := store.CreateRequest(db, rigRow.ID, cellRow.ID, "monitor", severity, priority, scope, string(payloadJSON))
		if reqErr != nil {
			return reqErr
		}
		fmt.Println("Created request from monitor failure")
		return nil

	default:
		return fmt.Errorf("unknown monitor subcommand: %s", op)
	}
}
