# GitHub Branch Protection for `main`

This is configured in GitHub repository settings (not in source code).

## Required Settings
1. Open `Settings -> Branches -> Branch protection rules -> Add rule`.
2. Branch name pattern: `main`.
3. Enable:
   - `Require a pull request before merging`
   - `Require approvals` (set to `1`)
   - `Dismiss stale pull request approvals when new commits are pushed` (recommended)
   - `Require linear history` (recommended with squash merge)
   - `Do not allow bypassing the above settings` (recommended)
4. Disable direct pushes to `main` for regular contributors.
5. Save rule.

## Merge Mode
- Use `Squash and merge`.
- PR title format: `<task-id>: short summary`.

## Verification
Try pushing directly to `main`:
- push must be rejected by GitHub.
- merge must be possible only via PR with approval.
