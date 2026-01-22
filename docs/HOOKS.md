# Hooks

Microforge generates `.claude/settings.json` inside each cell worktree.

- Stop hook -> `mforge hook stop`
  - Claims next queued assignment for active agent in SQLite
  - Writes an inbox mail file
  - Returns JSON `{ "continue": true, "reason": ... }` to force iterative continuation

- PreToolUse/PermissionRequest -> `mforge hook guardrails`
  - Blocks Write/Edit for reviewer/monitor/architect
  - Blocks builder writes outside scope (path-validated against the cell worktree)

If Claude Code prompts to approve hook config changes, approve them in `/hooks`.

## PATH for Hook Execution
Hooks run as shell commands; ensure `mforge` is on PATH in the hook environment. If hooks can’t find `mforge`, add a PATH export in your shell profile or wrap the hook command with an absolute path to `mforge`.
