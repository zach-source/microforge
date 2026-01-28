# Mforge Command Breakdown - This Session

## By Category

### Category: agent
- Count: 243
- Top Subcommands: spawn (43), wake (35), status (29), logs (29), create (28), relaunch (25)

### Category: epic
- Count: 93
- Top Subcommands: add-task (30), create (11), tree (8)

### Category: cell
- Count: 70
- Top Subcommands: bootstrap (27), add (23), agent-file (8)

### Category: assign
- Count: 52
- Top Subcommands: --task (34)

### Category: task
- Count: 51
- Top Subcommands: create (17), list (10)

### Category: round
- Count: 47
- Top Subcommands: review (21), start (14)

### Category: turn
- Count: 45
- Top Subcommands: status (13), start (8)

## Tmux (Agent Monitoring)

| Command       | Count | Purpose                |
|---------------|-------|------------------------|
| capture-pane  | 178   | Read agent output      |
| send-keys     | 100   | Send prompts to agents |
| list-sessions | 15    | Check running agents   |
| new-session   | 14    | Create agent sessions  |

## Key Patterns Observed
1. Heavy agent management - 243 agent commands (spawn, wake, status, logs)
2. Lots of monitoring - 178 tmux capture-pane calls to check agent progress
3. Manual intervention - 100 tmux send-keys to nudge stuck agents
4. Task assignment - 52 assign commands, mostly --task
5. Review workflow - 21 round review commands

## Improvement Opportunities

### Issue: Manual agent nudging
- Frequency: 100 send-keys
- Suggestion: Auto-polling daemon

### Issue: Checking agent status
- Frequency: 178 capture-pane
- Suggestion: Status dashboard / mforge watch

### Issue: Agent spawn without permissions
- Frequency: Multiple
- Suggestion: Default --dangerously-skip-permissions

### Issue: Manual inbox checking
- Frequency: Repeated
- Suggestion: Inbox polling in agent prompt

## Summary
The session was heavily focused on agent lifecycle management and manual monitoring,
suggesting mforge needs better autonomous agent supervision.
