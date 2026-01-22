# Hooks

Microforge generates `.claude/settings.json` inside each cell worktree.

- Stop hook -> `mf hook stop`
  - Claims next queued assignment for active agent in SQLite
  - Writes an inbox mail file
  - Returns JSON `{ "continue": true, "reason": ... }` to force iterative continuation

- PreToolUse/PermissionRequest -> `mf hook guardrails`
  - Blocks Write/Edit for reviewer/monitor
  - Blocks builder writes outside scope

If Claude Code prompts to approve hook config changes, approve them in `/hooks`.
