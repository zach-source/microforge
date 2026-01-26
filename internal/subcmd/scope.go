package subcmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/example/microforge/internal/rig"
)

func Scope(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge scope <list|show> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "list":
		if len(rest) != 1 {
			return fmt.Errorf("usage: mforge scope list <rig>")
		}
		rigName := rest[0]
		cells, err := rig.ListCellConfigs(home, rigName)
		if err != nil {
			return err
		}
		scopes := map[string][]string{}
		for _, cell := range cells {
			scope := strings.TrimSpace(cell.ScopePrefix)
			if scope == "" {
				continue
			}
			scopes[scope] = append(scopes[scope], cell.Name)
		}
		keys := make([]string, 0, len(scopes))
		for scope := range scopes {
			keys = append(keys, scope)
		}
		sort.Strings(keys)
		for _, scope := range keys {
			cells := scopes[scope]
			sort.Strings(cells)
			fmt.Printf("%s\t%s\n", scope, strings.Join(cells, ","))
		}
		return nil
	case "show":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge scope show <rig> --scope <path-prefix>")
		}
		rigName := rest[0]
		var scope string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--scope":
				if i+1 < len(rest) {
					scope = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(scope) == "" {
			return fmt.Errorf("--scope is required")
		}
		cells, err := rig.ListCellConfigs(home, rigName)
		if err != nil {
			return err
		}
		matched := make([]string, 0)
		for _, cell := range cells {
			if strings.TrimSpace(cell.ScopePrefix) == scope {
				matched = append(matched, cell.Name)
			}
		}
		sort.Strings(matched)
		for _, cell := range matched {
			fmt.Println(cell)
		}
		return nil
	default:
		return fmt.Errorf("unknown scope subcommand: %s", op)
	}
}
