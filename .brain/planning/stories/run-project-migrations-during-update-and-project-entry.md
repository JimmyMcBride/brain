---
created: "2026-04-15T23:32:00Z"
epic: release-install-and-update-flow
project: brain
spec: release-install-and-update-flow
status: done
title: Run Project Migrations During Update And Project Entry
type: story
updated: "2026-04-15T20:48:55Z"
---
# Run Project Migrations During Update And Project Entry

Created: 2026-04-15T23:32:00Z

## Description

Wire the project migration runner into the actual Brain upgrade lifecycle so the current repo upgrades during `brain update` and other older repos self-heal lazily on first use.


## Acceptance Criteria

- [x] `brain update --project <repo>` applies pending project migrations for the current Brain repo after the binary and skill refresh flow completes
- [x] `brain update --check` stays read-only and does not apply project migrations
- [x] App-backed Brain commands apply pending project migrations once per repo per process before project work begins, similar to the existing local-skill repair preflight
- [x] Commands that should stay mutation-free or bootstrap-only, such as `init`, `adopt`, `doctor`, `version`, `update`, and `skills`, skip the lazy migration preflight
- [x] Brain does not scan the machine for repos; older repos migrate only when the user updates that current project or later works in that repo directly


## Resources

- [[.brain/planning/specs/release-install-and-update-flow.md]]
- [[cmd/update.go]]
- [[cmd/root.go]]
- [[cmd/skill_refresh.go]]

## Notes

- Keep the sequencing explicit: binary update first, then skill refresh, then project migration for the current repo.
- Implemented with a hidden `brain context migrate` command for freshly installed binaries, a shared `projectMigrationRunner` for update orchestration, and a root-level project-repair preflight that reuses the new migration planner before app bootstrap.
