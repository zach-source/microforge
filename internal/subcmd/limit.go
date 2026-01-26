package subcmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/rig"
	"github.com/example/microforge/internal/turn"
)

func beadLimit(home, rigName, cellName, turnID string) error {
	limStr := strings.TrimSpace(os.Getenv("MF_BEAD_LIMIT_PER_TURN"))
	if limStr == "" {
		return nil
	}
	limit, err := strconv.Atoi(limStr)
	if err != nil || limit <= 0 {
		return fmt.Errorf("invalid MF_BEAD_LIMIT_PER_TURN: %s", limStr)
	}
	if strings.TrimSpace(cellName) == "" {
		return nil
	}
	if strings.TrimSpace(turnID) == "" {
		state, err := turn.Load(rig.TurnStatePath(home, rigName))
		if err != nil {
			return nil
		}
		turnID = strings.TrimSpace(state.ID)
	}
	if turnID == "" {
		return nil
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
	count := 0
	for _, issue := range issues {
		meta := beads.ParseMeta(issue.Description)
		if meta.Cell == cellName && meta.TurnID == turnID {
			count++
		}
	}
	if count >= limit {
		return fmt.Errorf("bead limit reached for %s (turn %s): %d/%d", cellName, turnID, count, limit)
	}
	return nil
}
