# Phase 1 — Requests Workflow

Goal: Add the missing monitor/reviewer request loop and manager triage.

## Database
- [x] Add `requests` table: id, rig_id, cell_id, source_role, severity, priority, scope_prefix, payload_json, status, created_at, updated_at.
- [x] Add indexes for `(rig_id, status)`, `(cell_id, status)`, `(priority, status)`.
- [x] Add models and CRUD helpers for requests.

## CLI
- [x] `mforge request create <rig> --cell <cell> --role <role> --severity <sev> --priority <p> --scope <path> --payload <json>`.
- [x] `mforge request list <rig> [--cell <cell>] [--status <status>] [--priority <p>]`.
- [x] `mforge request triage <rig> --request <id> --action create-task|merge|block`.
- [x] Validate required flags and print usage on missing args.

## Manager
- [x] Add triage logic: convert request → task(s) + assignment(s).
- [x] Support special request types: “monitor regression” and “reviewer follow-up”.
- [x] Define status transitions: `new → triaged → done/blocked`.

## Tests
- [x] Unit tests for request CRUD and status transitions.
- [x] CLI tests for create/list/triage.
- [x] Manager test: request triage generates task + assignment.

## Exit Criteria
- [x] Reviewer/monitor can generate requests and manager can triage into actionable builder work automatically.
