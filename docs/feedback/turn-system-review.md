# Mforge Turn System Review

Based on this session coordinating 4 builder agents across the Operations Security & Performance epic:

## What worked well

1. Task Distribution via Mail System - The inbox/outbox/archive pattern is clean and provides clear task handoff semantics.
2. Cell Scoping - Each cell having a dedicated worktree with its own scope (operator/, dataplane/, deploy/, blocks-service/) prevents conflicts and allows parallel work.
3. Agent Persistence - tmux sessions survive disconnects and agents can be monitored/nudged externally.
4. Context Compaction - Agents hitting context limits compacted and resumed work correctly (dataplane and blocks-service both compacted mid-task successfully).
5. Task Completion Output - The completion reports in mail/outbox/ provide good audit trail of what was done.

## Pain points observed

1. Message Processing Lag - Agents frequently showed messages echoed to prompt line but didn't process them until explicit Enter was sent. The input mode seems inconsistent.
2. "New tasks detected" Loop - All agents showed repeated "New tasks detected (N). Check mail/inbox..." notifications but often didn't autonomously start the next task. Required manual nudging.
3. No Task Claim/Lock Mechanism - When multiple tasks exist, there's no clear way for agents to "claim" a task to prevent duplicate work if multiple agents have overlapping scopes.
4. Agent State Opacity - Hard to tell from outside whether an agent is actively working, waiting for input, stuck at a prompt, or idle with no work.
5. Exit/Lifecycle Management - Telling idle agents to exit (/exit) didn't consistently work. Agents with empty inboxes sometimes just sat there.
6. Task Dependency - No built-in way to express "task B depends on task A" within the mail system.

## Suggested improvements

1. Agent Heartbeat/Status API
   - Example: `mforge status blocks --json` -> `{ "dataplane-builder": { "state": "working", "task": "ay00", "context_pct": 45 } }`
2. Auto-Advance Mode
   - Agent flag to automatically check inbox and start next task upon completion without manual nudge.
3. Task Locking
   - Add to task file header: `claimed_by`, `claimed_at`.
4. Idle Detection + Auto-Exit
   - If inbox empty for N seconds and no active work, agent gracefully exits.
5. Inter-Agent Messaging
   - File-based signals: `mail/signals/task-complete.json` -> `{ "task": "ay00", "agent": "dataplane-builder", "notify": ["blocks-service-builder"] }`
6. Wake Command Fix
   - `mforge wake` should reliably activate an idle agent; current tmux send-keys approach is fragile.

## Metrics from this session

| Metric | Value |
| --- | --- |
| Epic Tasks | 10 |
| Tasks Completed | 9 |
| Active Agents | 4 |
| Manual Nudges Required | ~15 |
| Context Compactions | 3 |
| Total Duration | ~2.5 hours |

## Bottom line

The system is functional for coordinating parallel work, but requires significant human supervision to keep agents moving. The mail-based task system is a good abstraction, but agents need better autonomous behavior around task pickup and completion signaling.

## Improvement plan

Phase 1 (stability + visibility)
- Add an agent heartbeat/status surface (CLI + JSON) backed by structured status events emitted from hooks.
- Add reliable follow/log streaming so state changes are visible without manual tmux capture.
- Improve wake reliability by auto-sending Enter for prompts and surfacing last input state.

Phase 2 (autonomy)
- Add auto-advance mode: after completion, agent scans inbox, claims next task, starts work without human prompt.
- Add a task claim/lock mechanism in mail metadata to prevent duplicate work.
- Add idle detection with configurable auto-exit and a supervisor policy to re-spawn as needed.

Phase 3 (coordination)
- Add task dependency metadata and resolver (blocked/unblocked) and surface in `mforge turn/round` status.
- Add inter-agent signaling (`mail/signals/`) for downstream notifications.
- Add guardrails so `mforge wake`/`assign` validate task ownership and status.

Phase 4 (UX cleanup)
- Provide a status dashboard/TUI view of agents, inbox depth, last activity, and current task.
- Add clear lifecycle commands: `mforge agent exit`, `mforge agent restart` with consistent behavior.
