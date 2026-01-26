# Repository Guidelines

## Project Structure & Module Organization
- `cmd/mforge/` holds the CLI entrypoint (`main` package) for the `mforge` binary.
- `internal/` contains the core packages (Beads client, tmux orchestration, hooks, task/agent logic). Keep new packages scoped and internal-only.
- `docs/` is for user-facing guides and design notes.
- `scripts/` contains helper scripts for local workflows.
- `Makefile` defines common dev commands.

## Build, Test, and Development Commands
- `make build` / `go build ./cmd/mforge`: build the `mforge` CLI binary.
- `make test` / `go test ./...`: run all Go tests.
- `make fmt` / `gofmt -w .`: format all Go files (required before PRs).

## Coding Style & Naming Conventions
- Use standard Go formatting (`gofmt`) with tabs for indentation.
- Package names are short, lowercase, and domain-specific (e.g., `queue`, `tmux`, `hooks`).
- Exported identifiers use `CamelCase`; unexported use `camelCase`.
- Keep CLI flags and subcommands consistent with existing `mforge` usage (see `README.md` examples).

## Testing Guidelines
- Use Go’s built-in `testing` package.
- Name test files `*_test.go` and match function names to the target (`TestQueueEnqueue`).
- Prefer table-driven tests for CLI behavior and edge cases.
- Run `go test ./...` before submitting changes.

## Commit & Pull Request Guidelines
- The repo currently has minimal history and no formal commit convention. Use clear, imperative subjects (e.g., “add task retry backoff”).
- PRs should include:
  - A concise description of the change and motivation.
  - Notes on tests run (e.g., `go test ./...`).
  - Screenshots or logs when behavior is CLI-visible.

## Agent-Specific Notes
- This project orchestrates long-lived agents via Beads + tmux. Keep changes deterministic and CLI-driven.
- Prefer extending existing workflows (`mforge task`, `mforge agent`, `mforge manager`) instead of adding parallel command paths.
- Bead creation can be rate-limited per cell/turn via `MF_BEAD_LIMIT_PER_TURN`.
