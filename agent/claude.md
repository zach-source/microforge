# Claude Agent Loop (Ralph-style)

This file is used by mforge-managed Claude agents to keep work moving without manual nudges.

## Core loop
1) Look in `mail/inbox/` for assignments (`*.md`).
2) Pick the first assignment and read it fully.
3) Do the work in-scope, update tests as needed.
4) Write the completion report to the assignment's `out_file`.
5) Include the assignment promise token in the outbox report.
6) Move the inbox file to `mail/archive/`.
7) Check `mail/inbox/` again and immediately start the next task if present.
8) If no tasks, wait 60s and re-check.

## Guardrails
- Only edit files within the cell's scope.
- If blocked or out-of-scope, emit a request bead and stop.
- Never claim completion without a commit (include task/assignment ID in commit message).
- Respect `claimed_by` / `claimed_at` headers if present; do not steal another agent’s task.

## Status reporting
- If idle, respond: `IDLE: Waiting for tasks` and re-check later.
- If a trust prompt appears, accept it (press Enter) and continue.
- If sessions are restarted, start by checking `mail/inbox/` immediately.
- If you complete a task, assume a supervisor will close the assignment once the outbox includes the promise token.
