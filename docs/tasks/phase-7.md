# Phase 7 — Reliability, Safety, CI

Goal: Make the system trustworthy on real monorepos.

## Beads Consistency
- [x] Ensure assignment selection respects dependencies and status transitions.
- [x] Define turn tagging conventions for Beads issues.

## Tests
- [x] Beads client tests for list/show/create/ready parsing.
- [x] Integration test: simulated worktree + outbox → manager reconciliation via Beads.

## CI + Lint
- [x] Add CI gate for `go test ./...`.
- [x] Add formatting + static checks.

## Exit Criteria
- [x] Core semantics are hard to break; CI catches regressions.
