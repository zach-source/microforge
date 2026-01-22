# Microforge — multi-role Claude Code controller (SQLite + tmux + hooks)

Microforge is a starter repo for controlling **Claude Code CLI** as a set of long-lived agents organized into **cells** (one cell per microservice / scope).

Each **cell** has:
- **Builder**: makes changes and pushes improvements for its service scope
- **Monitor**: watches tests/metrics/logs and emits “fix/enhance” tasks
- **Reviewer**: enforces quality + tool discipline; rejects corner-cutting
- (optional) **Architect**: updates docs, cross-service design, works with Manager
- **Manager** (global): triages tasks/requests and assigns work to cells/roles

Control surface:
- **SQLite** queue (tasks + assignments + requests)
- **tmux** sessions to “wake” sleeping agents via `tmux send-keys`
- **Claude Code hooks** to implement a Ralph-style iteration loop and guardrails

## Requirements
- Go 1.22+
- `tmux`
- `git` (optional but recommended)
- Claude Code CLI installed (`claude`)

## Build
```bash
go build ./cmd/mf
```

## Quickstart (one microservice cell)

1) Init a rig (points to your monorepo):
```bash
./mf init mono --repo ~/code/monorepo
```

2) Create a cell for a service scope (path prefix within repo):
```bash
./mf cell add mono payments --scope services/payments
./mf cell bootstrap mono payments
```

3) Spawn the agents (Claude sessions) in tmux:
```bash
./mf agent spawn mono payments builder
./mf agent spawn mono payments monitor
./mf agent spawn mono payments reviewer
```

4) Create a task and assign it to the cell’s builder:
```bash
./mf task create mono --title "Add /healthz endpoint" --body "Add handler + tests" --scope services/payments
./mf assign mono --task <task_id> --cell payments --role builder
```

5) Wake the builder (if it’s idle) and let hooks drive the loop:
```bash
./mf agent wake mono payments builder
```

6) Reconcile completion:
```bash
./mf manager tick mono --watch
```
