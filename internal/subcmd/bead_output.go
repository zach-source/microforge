package subcmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/example/microforge/internal/beads"
)

func printIssuesGrouped(issues []beads.Issue) {
	groups := map[string][]beads.Issue{}
	for _, issue := range issues {
		key := issue.Type + ":" + issue.Status
		groups[key] = append(groups[key], issue)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.SplitN(key, ":", 2)
		fmt.Printf("%s (%s)\n", parts[0], parts[1])
		for _, issue := range groups[key] {
			meta := beads.ParseMeta(issue.Description)
			line := fmt.Sprintf("- %s %s", issue.ID, issue.Title)
			if meta.Cell != "" {
				line += " (" + meta.Cell + ")"
			}
			fmt.Println(line)
		}
	}
}
