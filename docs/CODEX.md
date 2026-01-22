# Codex Runtime Notes

Microforge can run Claude Code or Codex per role via `rig.json`.

## Configure per-role runtime
Set `runtime_provider` to `codex` and override per-role commands as needed:

```json
{
  "runtime_provider": "codex",
  "runtime_cmd": "codex",
  "runtime_args": ["--resume"],
  "runtime_roles": {
    "manager": { "cmd": "codex", "args": ["--project", "manager"] },
    "architect": { "cmd": "codex", "args": ["--project", "architect"] }
  }
}
```

## Hook compatibility
Codex runs the same `mforge hook stop` and `mforge hook guardrails` commands. If your Codex environment uses a different hook mechanism, wrap the hook invocation so it still executes the `mforge` commands from the worktree.

## Approvals defaults
When using Codex, prefer conservative defaults (no writes outside scope; require explicit approvals for system-wide changes). Keep `AGENTS.md` and role identities in the worktree.
