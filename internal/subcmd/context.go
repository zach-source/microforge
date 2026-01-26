package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/example/microforge/internal/context"
	"github.com/example/microforge/internal/rig"
)

func Context(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge context <get|set|unset|list> ...")
	}
	op := args[0]
	rest := args[1:]
	switch op {
	case "get":
		state, err := context.Load(home)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No active rig")
				return nil
			}
			return err
		}
		if strings.TrimSpace(state.ActiveRig) == "" {
			fmt.Println("No active rig")
			return nil
		}
		fmt.Println(state.ActiveRig)
		return nil
	case "set":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge context set <rig>")
		}
		rigName := rest[0]
		if _, err := os.Stat(rig.RigConfigPath(home, rigName)); err != nil {
			return fmt.Errorf("unknown rig: %s", rigName)
		}
		state := context.State{ActiveRig: rigName}
		if err := context.Save(home, state); err != nil {
			return err
		}
		fmt.Printf("Active rig set to %s\n", rigName)
		return nil
	case "unset":
		if err := context.Clear(home); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		fmt.Println("Active rig cleared")
		return nil
	case "list":
		rigsDir := filepath.Join(home, "rigs")
		entries, err := os.ReadDir(rigsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No rigs")
				return nil
			}
			return err
		}
		state, _ := context.Load(home)
		active := strings.TrimSpace(state.ActiveRig)
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			marker := " "
			if name == active {
				marker = "*"
			}
			fmt.Printf("%s %s\n", marker, name)
		}
		return nil
	default:
		return fmt.Errorf("unknown context subcommand: %s", op)
	}
}
