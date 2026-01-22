# Phase 4 — Epics and Multi-Group Work

Goal: Support task graphs and cross-cell decomposition.

## Database
- [x] Add `epics` table: id, rig_id, title, body, status, created_at, updated_at.
- [x] Add `task_edges` or `task_links`: parent_task_id, child_task_id.
- [x] Add `epic_tasks` linking table (epic → tasks).

## CLI
- [x] `mforge epic create <rig> --title <t> [--body <md>]`.
- [x] `mforge epic add-task <rig> --epic <id> --task <id>`.
- [x] `mforge epic status <rig> --epic <id>` with rollups.
- [x] `mforge task split <rig> --task <id> --cells <a,b,c>` assistant helper.

## Manager
- [x] Enable multi-cell assignments for epic subtasks.
- [x] Define reviewer gating before merge/close.
- [x] Record conflicts and surface back to manager.

## Tests
- [x] Epic CRUD + rollup status tests.
- [x] Task graph integrity tests.

## Exit Criteria
- [x] One epic can drive multiple tasks across multiple cells in parallel.
