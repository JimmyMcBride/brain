# Migration from Standalone Plan

## Strategy

Treat Plan as reference implementation and compatibility source, not code to copy
line by line. Inventory, classify, and test behavior before extraction. Keep the
standalone repository and binary during migration.

## Rules

- preserve observable behavior before changing ownership
- separate Planning domain from CLI, filesystem, GitHub, Linear, and cloud
- retain `.plan/` for initial local migration
- share implementation between `plan` and `brain plan`
- version all machine-readable changes and test exit/stdout/stderr compatibility
- migrate with preview, backup, explicit confirmation where destructive, and
  idempotent reruns
- never silently change source ownership

## Mapping

| Current Plan boundary | Target |
| --- | --- |
| `internal/planning` mixed domain/application/adapters | Planning domain plus application services and adapters |
| `internal/workspace` `.plan/` schema and integration state | Local Planning storage adapter plus compatibility reader |
| `cmd/*` direct command construction | Shared Planning service called by native and wrapper CLIs |
| GitHub client/backend/reconcile | Optional GitHub companion adapter |
| Linear state/promotion | Standalone Plan cleanup; Brain refuses import |
| guided sessions | Planning workflow state over Core sessions/audit where appropriate |
| skills bundle | Brain agent-tool/module discovery direction |
| epics/stories | read/import compatibility mapping to initiatives/execution |

## Compatibility Checkpoints

1. Record golden workspace fixtures from supported Plan versions.
2. Record command/JSON/error/idempotency behavior.
3. Implement read-only detection in Brain Planning.
4. Run both CLIs against copied fixtures and compare.
5. Enable idempotent migration only after preview tests.
6. Keep a rollback/backup path and version diagnostics.
7. Deprecate wrapper behavior only with published support timing.

## Risks

- current `internal/planning` contains GitHub-specific application types
- `.plan/.meta/` combines schema, integration, migration, and guided state
- source-of-truth modes blur storage ownership and publication
- legacy epic/story and active spec execution overlap
- JSON payloads may be consumed by agents/scripts beyond documented use
- GitHub reconciliation depends on stable remote identity and idempotency
- standalone Plan must handle any Linear export before Brain migration begins

## Stale-Direction Audit

Repository search on 2026-07-27 found:

| Search concept | Remaining matches | Classification |
| --- | --- | --- |
| `Plan Cloud` / `plan-cloud` / `plan-cloud-sdk-go` | Planning vision and cloud-direction non-goals | Correct explicit retirement |
| official Linear integration | ADR index/ADR 0008 and exclusion contract | Correct superseded/exclusion decision |
| GitHub as permanent Planning source | None | No stale reference |
| Planning required by Brain | None | No stale reference |

Historical standalone Plan material may still describe shipped Linear and GitHub
behavior; the feature inventory classifies that implementation evidence as
retired or transitional rather than current Brain product direction.

See [Feature Inventory](plan-feature-inventory.md) for authoritative checklist.
