# Linear Exclusion and Standalone Plan Cleanup

## Decision

Brain Planning supports local, GitHub, and GitHub-hybrid migration sources only.
It never imports or implements Linear integration. Standalone Plan owns removal
or export of its historical Linear state. Do not preserve dead Linear concepts
inside Brain module APIs.

## Current Surface

| Surface | Current Plan location | Removal/migration action |
| --- | --- | --- |
| Source value `linear` | `internal/workspace/workspace.go`, `internal/planning/source_mode.go`, `cmd/source.go` | Reject new selection, diagnose old workspaces, then remove |
| Promotion target `linear` | `cmd/discuss.go`, collaboration/source-mode code | Remove official target and packet path |
| Team configuration | `LinearState` and `.plan/.meta/linear.json` | Standalone Plan removes/exports; Brain does not parse |
| Workspace schema | `Info.LinearFile`, surfaces/doctor/update logic | Remove from standalone Plan before shared extraction |
| Commands/help | source and promotion flags/messages | Remove/deprecate in compatibility stages |
| Agent/MCP handoff | guide/promotion payloads and skill/docs | Remove official instructions |
| Tests | workspace, source, collaboration, CLI tests | Preserve diagnostic fixtures; remove active behavior tests |
| Docs/roadmap | Plan README/docs/AGENTS/.plan brainstorms | Mark historical/superseded; remove current promise |
| Exported JSON | source state, promotion packet, guide packets | Version response; document removed fields/values |

## User-Safe Sequence

1. Architecture phase: exclude Linear from Brain Planning contracts and tracked
   Brain `.plan/` state.
2. Inventory affected commands, JSON fields, fixtures, and workspace versions.
3. At Brain Planning enablement, reject `source_mode: linear` without parsing or
   importing Linear integration metadata.
4. Provide a clear diagnostic: migrate workspace to local, GitHub, or hybrid
   using standalone Plan first.
5. Standalone Plan may provide a dry-run export preserving IDs/URLs/team data;
   Brain does not own that export.
6. Remove new Linear selection/promotion from standalone Plan.
7. Remove standalone packages/branches/types/tests/docs once compatibility policy permits.
8. Keep a stable unsupported-workspace error until the minimum supported schema no
   longer includes Linear.

## Error Contract

An old workspace receives an actionable unsupported-source message naming the
detected source mode and supported targets: local, GitHub, or GitHub hybrid. Brain
must not read, import, or silently reinterpret Linear-owned data.

## Explicit Non-Actions

This architecture PR does not modify standalone Plan code or rewrite historical
Plan artifacts. It removes Linear state from the Brain repository and excludes
Linear from every Brain Planning contract.
