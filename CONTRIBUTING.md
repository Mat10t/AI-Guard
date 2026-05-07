# Contributing Guide

## Scope
This workflow applies to all new tasks from this point forward.

## One-Time Local Setup
Run once in repository root:
- `make install-hooks`

## Mandatory Rule
`1 task = 1 branch = 1 PR`

## Task-First Start
1. Pick a task file in `tasks/` (for example `tasks/08-frontend/08-36-copy-anytime-and-hide-revoked-everywhere.md`).
2. Extract task id from filename (`08-36`).
3. Create a branch:
   - `feat/<task-id>-<slug>`
   - `fix/<task-id>-<slug>`

Examples:
- `feat/08-36-copy-anytime-hide-revoked`
- `fix/04-08-key-list-api-key-null-handling`

## Commit Rule
On `feat/*` and `fix/*` branches, commit message must start with:
- `<task-id>: ...`

Example:
- `08-36: hide revoked keys in API Keys table`

## Pull Request Rule
- One PR must cover only one task.
- PR must include:
  - link to task file in `tasks/...`
  - what was done (mapped to task checklist)
  - how validated (commands/screenshots/logs)
  - explicit out-of-scope
- Prefer `Squash and merge` with title:
  - `<task-id>: short summary`

## Merge Gate
Before merge, run:
- `make test`
- `make integration` (if backend/API affected)
- `make e2e` (if critical flows affected)
- `npm --prefix frontend run build` (if frontend affected)

## Task Metadata For Verification
In each new task file, fill service fields:
- `Branch:`
- `PR:`
- `Merged at:`

## Branch Protection (GitHub)
Configure protection for `main`:
- disallow direct pushes
- require pull request before merge
- require at least 1 approval
- restrict force pushes and deletions
