# Phase 6 — Runtime Providers

Goal: Support Claude Code and Codex under a shared orchestration model.

## Provider Abstraction
- [x] Add `runtime.provider` + per-role runtime command/args in rig config.
- [x] Implement provider-specific hook behavior if needed.

## Codex Startup
- [x] Add Codex boot workflow (AGENTS.md, approvals defaults).
- [x] Allow “Codex-as-manager” suggestions while MF remains source of truth.

## Tests
- [x] Config parsing tests for provider switching.
- [x] Smoke tests for provider command wiring.

## Exit Criteria
- [x] You can run Claude or Codex per role and behavior remains consistent.
