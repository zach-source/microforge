# Phase 9 — Remote tmux + Library MCP Agent

Goal: Support remote tmux sessions over SSH and add a shared Library agent (MCP server) for cross-service docs lookup.

## Remote tmux via SSH
- [x] Add rig config: `remote_host`, `remote_user`, `remote_tmux_prefix`, `remote_workdir`.
- [x] Add SSH helper: `mforge ssh <rig> --cmd <...>` to run remote commands.
- [x] Update `mforge agent spawn/stop/attach/wake/status` to optionally target remote tmux.
- [x] Add `mforge agent status --remote` to check remote session state.
- [x] Add tests for command construction and config parsing (mock SSH runner).

## Library Agent (MCP server)
- [x] Add `mforge library start <rig>` to launch the shared MCP server.
- [x] Provide a simple indexer for service docs paths (configurable list).
- [x] Add `mforge library query <rig> --service <name> --q <query>` for lookups.
- [x] Allow `mforge` agents to call the library MCP server for doc lookup.

## Context7 Integration
- [x] Allow the library agent to proxy to a Context7 MCP server when local docs are missing.
- [x] Add config for Context7 endpoint + auth.
- [x] Add tests for fallback logic.

## Exit Criteria
- [x] Agents can manage tmux sessions remotely over SSH.
- [x] All roles can query shared docs via the Library MCP server, with Context7 fallback.
