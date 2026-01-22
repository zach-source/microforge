# Phase 3 — Triad Behaviors

Goal: Make builder/monitor/reviewer real roles with guardrails and workflows.

## Builder
- [x] Enforce strict scope writes (path-based, optional “no shared/global without approval”).
- [x] Define builder tool policy (allowed commands, test commands, formatters).
- [x] Add builder policy checks to guardrails.

## Monitor
- [x] Add request emitters: run scoped tests on schedule or trigger.
- [x] Parse test output to generate requests with severity/priority.
- [x] Keep monitor read-only by default (guardrails enforced).

## Reviewer
- [x] Require tests for changes (request when missing).
- [x] Require docs updates for API behavior changes.
- [x] Enforce tool discipline (request follow-up on violations).

## Tests
- [x] Guardrails tests for builder policy enforcement.
- [x] Request generation tests for monitor/reviewer.

## Exit Criteria
- [x] Triad produces iterative improvements with clear guardrails and requests.
