package subcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/example/microforge/internal/beads"
	"github.com/example/microforge/internal/hooks"
)

func Hook(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge hook <stop|guardrails|emit> ...")
	}
	op := args[0]
	rest := args[1:]

	inBytes, _ := io.ReadAll(os.Stdin)
	var in hooks.ClaudeHookInput
	if len(inBytes) > 0 {
		_ = json.Unmarshal(inBytes, &in)
	}
	cwd := strings.TrimSpace(in.Cwd)
	if cwd == "" {
		wd, _ := os.Getwd()
		cwd = wd
	}

	switch op {
	case "stop":
		_ = rest // role is selected by agent wake/spawn via active-agent.json
		identity, err := hooks.LoadIdentityFromCWD(cwd)
		if err != nil {
			return err
		}
		client := beads.Client{RepoPath: identity.RepoPath}
		resp, err := hooks.StopHook(context.Background(), client, identity)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(resp)

	case "guardrails":
		identity, err := hooks.LoadIdentityFromCWD(cwd)
		if err != nil {
			return err
		}
		dec, err := hooks.GuardrailsHook(in, identity)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(dec)

	case "emit":
		if len(rest) < 2 {
			return fmt.Errorf("usage: mforge hook emit --event <name>")
		}
		event := ""
		for i := 0; i < len(rest); i++ {
			if rest[i] == "--event" && i+1 < len(rest) {
				event = rest[i+1]
				i++
			}
		}
		if strings.TrimSpace(event) == "" {
			return fmt.Errorf("--event is required")
		}
		identity, err := hooks.LoadIdentityFromCWD(cwd)
		if err != nil {
			return err
		}
		payload := map[string]any{}
		if len(inBytes) > 0 {
			_ = json.Unmarshal(inBytes, &payload)
		}
		return hooks.DispatchHook(event, payload, identity)

	default:
		return fmt.Errorf("unknown hook subcommand: %s", op)
	}
}
