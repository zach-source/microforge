# Reporting

Microforge provides lightweight status summaries for backlog health.

## Summary command

```bash
mforge report <rig> [--cell <cell>]
```

Output includes:
- Request counts by status.
- Task counts by status.
- Oldest request/task timestamps to indicate aging.

If `--cell` is supplied, request counts filter to that cell and tasks filter to the cell scope prefix.
