# Release Checklist

## Pre-Release
- Run formatting: `make fmt`
- Run tests: `make test` (or `go test ./...`)
- Ensure README quickstart works on a fresh machine
- Confirm hooks config is generated correctly (`.claude/settings.json`)
- Verify tmux workflows: spawn, wake, stop

## Versioning
- Update `CHANGELOG.md` with release notes
- Tag the release: `git tag vX.Y.Z`
- Push tags: `git push --tags`

## Post-Release
- Smoke test the CLI using a new rig/cell
- File any follow-up tasks for backlog improvements
