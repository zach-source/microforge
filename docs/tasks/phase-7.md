# Phase 7 — Reliability, Safety, CI

Goal: Make the system trustworthy on real monorepos.

## SQLite Correctness
- [x] Improve claim/lock semantics to avoid double-claim.
- [x] Add migrations framework with versioned migrations.

## Tests
- [x] DB unit tests for claims, requests, and task graph rollups.
- [x] Integration test: simulated worktree + outbox → manager reconciliation.

## CI + Lint
- [x] Add CI gate for `go test ./...`.
- [x] Add formatting + static checks.

## Exit Criteria
- [x] Core semantics are hard to break; CI catches regressions.
