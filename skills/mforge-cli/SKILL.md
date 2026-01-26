---
name: mforge-cli
description: Use for Microforge (mforge) CLI workflows: rigs/context, agent creation/bootstrap, epic planning, rounds/engine runs, beads tasks/assignments, and observability.
---

# Mforge CLI Skill

Use this skill when the task involves Microforge (mforge) workflows or requires correct CLI command usage.

## Preconditions
- Prefer an active rig context; most commands assume `mforge context set <rig>` has been run.
- Mailboxes == Beads (tasks/assignments are Beads issues; inbox/outbox live under each cell worktree).

## Core flows

### Rig + context
```bash
mforge init <rig> --repo <path>
mforge context set <rig>
```

### Agent lifecycle
```bash
mforge agent create <path> --description "<text>" [--class crew|worker]
mforge agent bootstrap <name>
mforge agent spawn <cell> <role>
mforge agent wake <cell> <role>
mforge agent logs <cell> <role> --follow
mforge agent logs --follow --all
mforge agent status --cell <cell> --role <role> --json
```

### Epic planning + status
```bash
mforge epic create --title "<title>" --short-id <id>
mforge epic design <id|short-id>
mforge epic tree <id|short-id>
```

### Turn + round orchestration
```bash
mforge round start --wait
mforge round review --wait
mforge round review --all
mforge round review --changes-only
mforge round review --base <branch>
mforge round merge --feature <branch> [--base <branch>]
mforge checkpoint --message "round N"
mforge convoy start --epic <id> [--role <role>] [--title "<text>"]
```

### Supervisor
```bash
mforge watch --interval 60
mforge watch --role builder
mforge watch --fswatch
```

### Engine loop
```bash
mforge engine run --rounds <n>
mforge engine run --completion-promise "<text>"
mforge engine emit --type <EventType> --scope <path> --title "<title>" --payload '<json>'
```

### Task decomposition
```bash
mforge task decompose --task <id> --titles "step a,step b" [--kind plan]
```

### Migrations
```bash
mforge migrate beads --all
```

## Behavior notes
- Assignments are created as Beads issues of type `assignment` and linked with `related:<task_id>`.
- A round start assigns unassigned tasks to cells based on scope, then wakes the appropriate role.
- A round review defaults to changes-only (skips cells without git diffs); use `--all` to force reviews for every cell.
- A round review auto-spawns reviewer/cell agents before waking them.
- Round merge aggregates all cell branches into a feature branch.
- Convoys assign all epic tasks and wake relevant agents.
- Agent spawn/wake/relaunch refuses non-bootstrapped cells; assignment/round paths auto-bootstrap.
- Agent spawn preflight checks for worktree code + write access (skip only with `MF_ALLOW_EMPTY_WORKTREE=1` or test envs).
- Assignments only auto-close when an outbox promise is met *and* the latest git commit message includes the assignment or related task ID.
- `mforge watch` nudges idle agents when inbox has tasks; `--fswatch` uses filesystem events if installed.

## When unsure
- Use `mforge scope list` and `mforge scope show --scope <path>` to confirm scope mappings.
- Use `mforge bead list --type task` to inspect outstanding tasks and dependencies.
