package subcmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
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
	cfg, err := rig.LoadRigConfig(rig.RigConfigPath(home, rigName))
	if err != nil {
		return err
	}
	client := beads.Client{RepoPath: cfg.RepoPath}
	issues, err := client.List(nil)
	if err != nil {
		return err
	}

	reqCounts := map[string]int{}
	taskCounts := map[string]int{}
	oldReq := ""
	oldTask := ""
	for _, issue := range issues {
		meta := beads.ParseMeta(issue.Description)
		if strings.TrimSpace(cellName) != "" && meta.Cell != cellName {
			continue
		}
		switch strings.ToLower(issue.Type) {
		case "request", "observation":
			reqCounts[issue.Status]++
			if issue.CreatedAt != "" && (oldReq == "" || issue.CreatedAt < oldReq) {
				oldReq = issue.CreatedAt
			}
		case "task", "assignment":
			taskCounts[issue.Status]++
			if issue.CreatedAt != "" && (oldTask == "" || issue.CreatedAt < oldTask) {
				oldTask = issue.CreatedAt
			}
		}
	}

	fmt.Println("Requests")
	printCounts(reqCounts)
	if oldReq != "" {
		fmt.Printf("oldest_request\t%s\n", oldReq)
	}
	fmt.Println("Tasks")
	printCounts(taskCounts)
	if oldTask != "" {
		fmt.Printf("oldest_task\t%s\n", oldTask)
	}
	return nil
}

func printCounts(counts map[string]int) {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s\t%d\n", k, counts[k])
	}
}
