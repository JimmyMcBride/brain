# Official Linear Removal

## Decision

Official Linear integration is retired. A future community module may integrate
Linear; Brain and official Planning make no commitment to it. Do not preserve
dead Linear concepts inside generic provider APIs.

## Current Surface

| Surface | Current Plan location | Removal/migration action |
| --- | --- | --- |
| Source value `linear` | `internal/workspace/workspace.go`, `internal/planning/source_mode.go`, `cmd/source.go` | Reject new selection, diagnose old workspaces, then remove |
| Promotion target `linear` | `cmd/discuss.go`, collaboration/source-mode code | Remove official target and packet path |
| Team configuration | `LinearState` and `.plan/.meta/linear.json` | Read-only migration diagnostic/export |
| Workspace schema | `Info.LinearFile`, surfaces/doctor/update logic | Stop creating after compatibility window |
| Commands/help | source and promotion flags/messages | Remove/deprecate in compatibility stages |
| Agent/MCP handoff | guide/promotion payloads and skill/docs | Remove official instructions |
| Tests | workspace, source, collaboration, CLI tests | Preserve diagnostic fixtures; remove active behavior tests |
| Docs/roadmap | Plan README/docs/AGENTS/.plan brainstorms | Mark historical/superseded; remove current promise |
| Exported JSON | source state, promotion packet, guide packets | Version response; document removed fields/values |

## User-Safe Sequence

1. Architecture phase: mark official direction retired; make no code removal.
2. Inventory affected commands, JSON fields, fixtures, and workspace versions.
3. Add a detector for `source_of_truth: linear`, `promotion_target:
   linear_issue`, or nonempty `.plan/.meta/linear.json`.
4. Provide a clear diagnostic: official Linear support ended; switch ownership to
   local/GitHub during transition or export metadata for a community adapter.
5. Add a dry-run migration that preserves IDs/URLs/team data in an archive or
   documented export; never silently discard it.
6. Remove new Linear selection/promotion while retaining read-only detection.
7. Remove packages/branches/types/tests/docs once compatibility policy permits.
8. Keep a stable unsupported-workspace error until the minimum supported schema no
   longer includes Linear.

## Error Contract

An old workspace must receive an actionable unsupported-configuration message
including detected path/value, last supported Plan version when known, non-
destructive migration command, backup/export location, and supported target
modes. Brain must not silently reinterpret Linear-owned data as local.

## Explicit Non-Actions

This architecture PR does not delete Linear code, rewrite old historical
brainstorms, or invent an official generic tracker interface for Linear parity.
