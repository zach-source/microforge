# Phase 2 — Ralph-Style Loop

Goal: Agents sleep at the prompt but continue work when assignments exist.

## Stop Hook Semantics
- [x] Ensure `mforge hook stop` claims next assignment, injects mail content, and forces continuation.
- [x] Add a clean idle response when no assignments exist (no spin).
- [x] Validate in-worktree deterministic behavior for all roles.

## Completion Detection
- [x] Standardize completion promise token per assignment (default `DONE`).
- [x] Ensure manager reconciles task + assignment to done when token appears.
- [x] Add optional archive move for inbox/outbox when done.

## Tests
- [x] Stop hook test: no assignment → Continue=false.
- [x] Stop hook test: assignment claimed → inbox written + Continue=true.
- [x] Manager test: promise found → task/assignment done.
- [x] Manager test: missing outbox → no updates.

## Exit Criteria
- [x] Waking a tmux session once is enough; stop hook drives until completion then idles.
