package subcmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/microforge/internal/store"
)

func Architect(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge architect <docs|contract|design> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "docs":
		return architectRequest(home, rest, "Docs update required")
	case "contract":
		return architectRequest(home, rest, "Contract check required")
	case "design":
		return architectRequest(home, rest, "Design review required")
	default:
		return fmt.Errorf("unknown architect subcommand: %s", op)
	}
}

func architectRequest(home string, args []string, title string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge architect <docs|contract|design> <rig> --cell <cell> --details <text> [--scope <path>]")
	}
	rigName := args[0]
	var cellName, details, scope string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--cell":
			if i+1 < len(args) {
				cellName = args[i+1]
				i++
			}
		case "--details":
			if i+1 < len(args) {
				details = args[i+1]
				i++
			}
		case "--scope":
			if i+1 < len(args) {
				scope = args[i+1]
				i++
			}
		}
	}
	if strings.TrimSpace(cellName) == "" || strings.TrimSpace(details) == "" {
		return fmt.Errorf("--cell and --details are required")
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
	payload := map[string]any{
		"title":   title,
		"details": details,
		"scope":   scope,
	}
	payloadJSON, _ := json.Marshal(payload)
	if _, err := store.CreateRequest(db, rigRow.ID, cellRow.ID, "architect", "med", "p2", scope, string(payloadJSON)); err != nil {
		return err
	}
	fmt.Println("Created architect request")
	return nil
}
