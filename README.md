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
go build ./cmd/mforge
```

## Install
```bash
go install ./cmd/mforge
```

## Nix
```bash
nix build
```

## Getting Started

1) Build and ensure `mforge` is on your PATH:
```bash
go build ./cmd/mforge
export PATH="$PWD:$PATH"
```

2) Initialize a rig and create a cell:
```bash
./mforge init mono --repo ~/code/monorepo
./mforge cell add mono payments --scope services/payments
./mforge cell bootstrap mono payments
```

3) Spawn and wake the builder:
```bash
./mforge agent spawn mono payments builder
./mforge agent wake mono payments builder
```

4) Create work and assign it:
```bash
./mforge task create mono --title "Add /healthz endpoint" --body "Add handler + tests" --scope services/payments
./mforge assign mono --task <task_id> --cell payments --role builder
```

5) Monitor progress and reconcile:
```bash
./mforge manager tick mono --watch
./mforge report mono --cell payments
```

Optional:
- Remote tmux: set `remote_host`, `remote_user`, `remote_port`, and `remote_workdir` in `rig.json`, then pass `--remote` to `mforge agent` commands.
- Library MCP: set `library_docs` and `library_addr` in `rig.json`, then run `./mforge library start mono`.

## Quickstart (one microservice cell)

1) Init a rig (points to your monorepo):
```bash
./mforge init mono --repo ~/code/monorepo
```

2) Create a cell for a service scope (path prefix within repo):
```bash
./mforge cell add mono payments --scope services/payments
./mforge cell bootstrap mono payments
# optional architect role
./mforge cell bootstrap mono payments --architect
```

3) Spawn the agents (Claude sessions) in tmux:
```bash
./mforge agent spawn mono payments builder
./mforge agent spawn mono payments monitor
./mforge agent spawn mono payments reviewer
```

4) Create a task and assign it to the cell’s builder:
```bash
./mforge task create mono --title "Add /healthz endpoint" --body "Add handler + tests" --scope services/payments
./mforge assign mono --task <task_id> --cell payments --role builder
```

5) Wake the builder (if it’s idle) and let hooks drive the loop:
```bash
./mforge agent wake mono payments builder
```

6) Reconcile completion:
```bash
./mforge manager tick mono --watch
```
