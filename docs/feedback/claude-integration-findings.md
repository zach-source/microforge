# Claude CLI Integration Test Findings

Date: 2026-01-28
Test: `scripts/integration/claude_turn_test.sh`

## Goal
Validate the full turn loop with **real Claude CLI**:
- create rig + cell
- spawn agent
- create task + assignment + inbox
- agent completes task + commits
- manager closes assignment
- completion signal emitted

## Results (latest run)
- ✅ Agent spawned and accepted wake prompt via tmux
- ✅ Task completed, outbox promise detected
- ✅ Commit present with task id in message
- ✅ Manager tick closed assignment
- ✅ Completion signal file created

## Issues encountered and fixes applied
1) **Claude settings.json format invalid**
   - Symptom: Claude showed “permissions: Expected object, but received array”.
   - Fix: Update generated `.claude/settings.json` to use object form:
     - `"permissions": { "allow": ["Bash", "Read", "Write", "Edit"] }`
   - Files: `internal/subcmd/cell.go`, `internal/subcmd/migrate.go`

2) **Agent stuck at resume prompt**
   - Symptom: “No conversations found to resume” when launching new agent sessions.
   - Fix: Remove `--resume` from default runtime args; keep only `--dangerously-skip-permissions`.
   - Files: `internal/rig/rig.go`, `internal/subcmd/migrate.go`

3) **tmux send-keys target invalid (window 0 not found)**
   - Symptom: `can't find window: 0` when nudging agents.
   - Fix: send-keys target now uses session only (no `:0.0`).
   - File: `internal/subcmd/agent.go`

4) **Assignment commit check failed when deps missing**
   - Symptom: assignment stayed `in_progress` even with commit.
   - Cause: Beads JSON doesn’t expose `deps`, so commit lookup by task id failed.
   - Fix: fall back to parsing task id from assignment title.
   - File: `internal/subcmd/manager.go`

5) **Assignment id not exposed**
   - Symptom: integration harness couldn’t correlate assignment from CLI output.
   - Fix: `mforge assign` now prints assignment id.
   - File: `internal/subcmd/assign.go`

## Test harness details
- Script: `scripts/integration/claude_turn_test.sh`
- Uses a temporary repo + temporary MF_HOME
- Spawns real Claude session in tmux
- Verifies outbox + assignment closure + completion signal

## Remaining observations
- `turn status` still shows `Tasks: 1 open` after assignment completion (expected, tasks are not auto-closed).
- `turn status` summary shows 0 commits but “By Cell” includes 1 commit (branch vs main mismatch).

## Next improvements to consider
- Add a `mforge task complete --auto` option to close task when assignment completes.
- Improve `turn status` commit counts to include cell branch activity.
- Add `mforge integration claude` command to run the harness in a structured way.
