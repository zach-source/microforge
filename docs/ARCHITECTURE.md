# Architecture: cells + roles + manager + architect

A **cell** is a unit focused on one microservice scope (e.g., `services/payments`) in a monorepo.

A cell has a shared git worktree and three roles:
- builder
- monitor
- reviewer

A global **manager** triages tasks and assigns them to cell roles.
An optional **architect** keeps docs and cross-service interfaces coherent.

Cells can also run as a **single agent** (merged triad) using the `cell` role.
Role behavior is defined by editable files under `.mf/roles/` in each worktree.

The **engine** and **round** commands orchestrate deterministic phases
(intake → plan → execute → review) using Beads events and assignments.

The “Ralph-style loop” here is implemented using a Stop hook that claims ready assignment beads and injects them into Claude’s context.
