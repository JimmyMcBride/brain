---
created: "2026-04-15T23:30:00Z"
epic: release-install-and-update-flow
project: brain
spec: release-install-and-update-flow
status: done
title: Add Project Migration Ledger And Planner
type: story
updated: "2026-04-15T20:36:21Z"
---
# Add Project Migration Ledger And Planner

Created: 2026-04-15T23:30:00Z

## Description

Add a repo-local migration ledger and planning model so Brain can track which soft project migrations have been applied independently from the installed skill manifest or raw Brain semver.


## Acceptance Criteria

- [x] Brain stores project migration state under `.brain/state/` with a stable schema version, applied migration ids, and run metadata suitable for future migrations
- [x] Migration planning uses named migration ids or steps instead of only comparing the current Brain version string
- [x] Repos that do not already use Brain are detected and skipped cleanly without creating new workspace state
- [x] Invalid or missing migration state is treated as recoverable and does not require manual state-file surgery before Brain can proceed


## Resources

- [[.brain/planning/specs/release-install-and-update-flow.md]]
- [[cmd/update.go]]
- [[cmd/root.go]]
- [[internal/projectcontext/manager.go]]

## Notes

- Keep the project migration ledger separate from `.brain-skill-manifest.json`. Skill freshness and project migration state are related, but they are not the same lifecycle.
- Implemented in `internal/projectcontext/migrations.go` with focused coverage in `internal/projectcontext/migrations_test.go`.
