# Phase 0 — MVP Hardening

Goal: Make the current skeleton compile cleanly, run cleanly, and be safe to iterate on.

## Repo Baseline
- [x] Verify repo path at `~/repos/workspaces/microforge` and build succeeds: `go build ./cmd/mforge`.
- [x] Add a DB smoke test that opens a DB, migrates, and validates core tables exist.
- [x] Add a smoke test for `mforge init` + `mforge cell bootstrap` to ensure paths and config are created.

## Hook Plumbing Correctness
- [x] Validate `mforge` binary is on PATH in hook execution context; document how to set PATH for hooks.
- [x] Ensure `mforge hook stop` uses only `active-agent.json` in the worktree (no external state).
- [x] Ensure `mforge hook guardrails` deterministically evaluates paths within worktree.

## Active Agent Identity Switching
- [x] Make `mforge agent wake` and `mforge agent spawn` update `active-agent.json` atomically for the target role.
- [x] Add a test that switching roles results in a predictable `active-agent.json` payload.

## tmux Lifecycle
- [x] Confirm `mforge agent spawn/stop/attach/wake` are idempotent (safe to repeat).
- [x] Add `mforge agent status` to show tmux session presence + last activity timestamp.
- [x] Add optional tmux integration test gated by build tag and env var.

## Exit Criteria
- [x] Init → cell bootstrap → spawn triad → assign → wake → outbox → manager marks done without manual file edits.
