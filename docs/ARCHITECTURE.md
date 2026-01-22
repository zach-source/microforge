# Architecture: cells + roles + manager + architect

A **cell** is a unit focused on one microservice scope (e.g., `services/payments`) in a monorepo.

A cell has a shared git worktree and three roles:
- builder
- monitor
- reviewer

A global **manager** triages tasks and assigns them to cell roles.
An optional **architect** keeps docs and cross-service interfaces coherent.

The “Ralph-style loop” here is implemented using a Stop hook that claims queued work from SQLite and injects it into Claude’s context.
