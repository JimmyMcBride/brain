---
created: "2026-04-15T23:31:00Z"
epic: release-install-and-update-flow
project: brain
spec: release-install-and-update-flow
status: todo
title: Implement First-Wave Project Soft Migrations
type: story
updated: "2026-04-15T23:31:00Z"
---
# Implement First-Wave Project Soft Migrations

Created: 2026-04-15T23:31:00Z

## Description

Implement the initial set of automatic soft migrations for existing Brain repos by reusing safe, idempotent Brain-managed operations instead of inventing a second document mutation system.


## Acceptance Criteria

- [ ] First-wave migrations refresh Brain-managed generated context surfaces such as `AGENTS.md`, `.brain/context/*`, and generated project docs without clobbering local notes outside managed blocks
- [ ] Existing agent files with Brain-managed integration blocks or legacy Brain wrapper blocks are updated or migrated in place without creating missing agent files
- [ ] Automatic migrations do not create a Brain workspace in non-Brain repos and do not mutate unmanaged agent files that have no Brain-managed content yet
- [ ] Re-running the same project migrations is idempotent and reports `unchanged` or equivalent no-op behavior once the repo is current


## Resources

- [[.brain/planning/specs/release-install-and-update-flow.md]]
- [[cmd/context.go]]
- [[cmd/adopt.go]]
- [[internal/projectcontext/manager.go]]

## Notes

- Prefer calling existing `context refresh` and agent-integration sync primitives under the hood instead of maintaining a parallel set of migration-specific markdown writers.
