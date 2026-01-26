# Microforge — multi-role Claude Code controller (Beads + tmux + hooks)

Microforge is a starter repo for controlling **Claude Code CLI** as a set of long-lived agents organized into **cells** (one cell per microservice / scope).

Each **cell** has:
- **Builder**: makes changes and pushes improvements for its service scope
- **Monitor**: watches tests/metrics/logs and emits “fix/enhance” tasks
- **Reviewer**: enforces quality + tool discipline; rejects corner-cutting
- (optional) **Architect**: updates docs, cross-service design, works with Manager
- **Manager** (global): triages tasks/requests and assigns work to cells/roles

Control surface:
- **Beads** issue bus (tasks + assignments + requests)
- **tmux** sessions to “wake” sleeping agents via `tmux send-keys`
- **Claude Code hooks** to implement a Ralph-style iteration loop and guardrails

## Requirements
- Go 1.22+
- `tmux`
- `git` (optional but recommended)
- Claude Code CLI installed (`claude`)
- Beads CLI (`bd`) installed and available on PATH

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

2) Initialize a rig, set context, and create a cell:
```bash
mforge init mono --repo ~/code/monorepo
mforge context set mono
mforge cell add payments --scope services/payments
mforge cell bootstrap payments
mforge cell bootstrap payments --single
```
Note: an active rig context is required for all non-`mforge rig` commands.

Rig lifecycle:
```bash
mforge rig list
mforge rig backup mono
mforge rig rename mono mono-new
mforge rig restore ~/.microforge/backups/rig-mono-*.tar.gz --name mono
mforge rig delete mono
```

3) Spawn and wake the builder:
```bash
mforge agent spawn payments builder
mforge agent wake payments builder
```

4) Create work and assign it:
```bash
mforge task create --title "Add /healthz endpoint" --body "Add handler + tests" --scope services/payments
mforge assign --task <task_id> --cell payments --role builder
```

5) Monitor progress and reconcile:
```bash
mforge manager tick --watch
mforge report --cell payments
```

Optional:
- Start a turn + bead triage helpers:
```bash
mforge turn start
mforge bead list --type request
mforge bead triage --id <bead_id> --cell payments --role builder
mforge turn slate
```
- Review + PR workflow:
```bash
mforge review create --title "Review payments work" --cell payments
mforge pr create --title "payments: /healthz" --cell payments --url <pr_url>
mforge pr link-review <pr_id> <review_id>
mforge pr ready <pr_id>
mforge merge run --as merge-manager
```
- Remote tmux: set `remote_host`, `remote_user`, `remote_port`, and `remote_workdir` in `rig.json`, then pass `--remote` to `mforge agent` commands.
- Library MCP: set `library_docs` and `library_addr` in `rig.json`, then run `./mforge library start`.
- Per-turn bead rate limit: set `MF_BEAD_LIMIT_PER_TURN=25` to cap beads per cell per turn.
- Engine events require Beads custom types. Add to `.beads/config.yaml`:
```yaml
types.custom: "event,turn,assignment,request,observation,decision,contract,review,pr,build,deploy,doc"
```
Or run:
```bash
mforge migrate beads --all
```

## Turn-Based Flow (Recommended)

1) Start a turn and auto-assign work:
```bash
mforge turn run --role builder
```

2) Wait for all assignments to finish:
```bash
mforge wait
```

3) Mark PRs ready + sync the coordinator:
```bash
mforge pr ready <pr_id>
mforge coordinator sync
```

4) Merge and end the turn:
```bash
mforge merge run --as merge-manager
mforge turn end
```

## Epic + Round Flow (Managers + Agents)

```bash
mforge init my-rig --repo .
mforge context set my-rig

mforge agent create ./operator --description "a golang operator that deploys all our resource"
mforge agent bootstrap operator
# bootstrap more agents...

mforge epic create --title "add auth using ory" --short-id auth
mforge epic design auth
mforge epic tree

mforge round start --wait
mforge round review --wait
mforge checkpoint --message "round 1"
```

Engine automation:
```bash
mforge engine run --rounds 3
mforge engine run --completion-promise "epic auth is done"
```

## Engine (Event-Driven Turns)

The engine CLI runs a simple Intake → Plan → Execute → Review loop backed by Beads events.

1) Emit an event:
```bash
mforge engine emit --type FeatureRequest --scope services/payments --title "Add /healthz" \
  --payload '{"scope":"services/payments","title":"Add /healthz"}'
```

2) Run the engine (plan tasks + wake agents):
```bash
mforge engine run --wait
```

3) Inspect or drain events:
```bash
mforge engine drain
```

## Observability (Logs + Heartbeats)

Each agent writes logs and heartbeats under:
```
~/.microforge/rigs/<rig>/agents/<cell>/<role>/
```

Commands:
```bash
mforge agent logs payments builder --lines 200
mforge agent logs payments builder --follow
mforge agent heartbeat payments builder
mforge agent status --cell payments --role builder --json
```

TUI dashboard:
```bash
mforge tui --interval 2
```

## Quickstart (one microservice cell)

1) Init a rig (points to your monorepo):
```bash
mforge init mono --repo ~/code/monorepo
mforge context set mono
```

2) Create a cell for a service scope (path prefix within repo):
```bash
mforge cell add payments --scope services/payments
mforge cell bootstrap payments
# optional architect role
mforge cell bootstrap payments --architect
# optional single-agent per cell (merged triad)
mforge cell bootstrap payments --single
```

3) Spawn the agents (Claude sessions) in tmux:
```bash
mforge agent spawn payments builder
mforge agent spawn payments monitor
mforge agent spawn payments reviewer
mforge agent spawn payments cell
```

4) Create a task and assign it to the cell’s builder:
```bash
mforge task create --title "Add /healthz endpoint" --body "Add handler + tests" --scope services/payments
mforge assign --task <task_id> --cell payments --role builder
```
Optional task updates:
```bash
mforge task complete --task <task_id> --reason "done"
mforge task delete --task <task_id> --dry-run
```

5) Wake the builder (if it’s idle) and let hooks drive the loop:
```bash
mforge agent wake payments builder
```

6) Reconcile completion:
```bash
mforge manager tick --watch
```

Scope helpers:
```bash
mforge scope list
mforge scope show --scope services/payments
```

Role guides (editable):
```
~/.microforge/rigs/<rig>/cells/<cell>/worktree/.mf/roles/
```

## Shell completion (bash/zsh + fzf)

Enable completions by sourcing the script for your shell:

```bash
# bash
source scripts/completions/mforge.bash
```

```zsh
# zsh
source scripts/completions/mforge.zsh
```

Or use the installer (auto-detects bash/zsh):

```bash
source scripts/completions/install.sh
```

Or let `mforge` print the exact source command:

```bash
$(mforge completions install)
```

If `fzf` is installed, completions will open an fzf picker by default. To disable
fzf and use standard tab completion, set:

```bash
export MF_FZF_DISABLE=1
```
