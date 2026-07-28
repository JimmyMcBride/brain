# Planning Module Implementation Roadmap

## Phase 0 — Architecture and Inventory

This documentation phase establishes product boundaries, feature classification,
module/security contracts, ADRs, storage modes, GitHub transition, Linear
exclusion, CLI compatibility, Brain knowledge loop, and cloud direction. It
contains no major migration code.

Exit: docs are internally consistent, Brain/Plan implementations are inventoried,
context is refreshed, validation passes, and unresolved questions are explicit.

## Phase 1 — Minimal Internal Module Framework

Status: Implemented by
`.plan/specs/minimal-internal-module-framework.md`; pending review and merge.

Deliver module descriptor/interface, registry, capability declarations,
project-scoped enable/disable state, namespaced configuration, permissions,
minimal lifecycle, health, tests, and one test-only reference module.

Do not load external community processes. Do not migrate Planning domain code.

## Phase 2 — Planning Domain Extraction

Deliver storage-neutral brainstorm, spec, initiative, roadmap, readiness,
approval, execution, queue, and runtime-slice models with unit tests. Keep Cobra,
filesystem, GitHub, and cloud dependencies outside domain packages. Do not
extract Linear code or types.

## Phase 3 — Local Planning Module

Deliver module enablement, native command group, `.plan/` read/write compatibility,
workspace detection/migration, Brain context access, Planning events/permissions,
and local tests.

## Phase 4 — Plan CLI Compatibility

Deliver wrapper/shared implementation, command mapping, warning policy, docs, and
script/exit/JSON compatibility tests.

## Phase 5 — GitHub Adapter Transition

Move retained GitHub collaboration/publication/repository/execution behavior
behind generic Planning ports. Support planning PRs and retained Discussion,
issue, milestone, Project, and reconciliation behavior. No GitHub type remains in
Planning domain.

## Phase 6 — Standalone Plan Linear Cleanup

Complete Linear removal in standalone Plan before shared implementation crosses
into Brain. Brain Planning accepts only `local`, `github`, and `hybrid`
workspaces. A Linear-configured workspace receives standalone Plan migration
guidance; no Linear schema, adapter, command, permission, or event enters Brain.

## Phase 7 — Brain Planning Knowledge Loop

Deliver context requests, Brain source references, Planning context packets,
completion memory proposals, provenance/freshness, and permission tests.

## Phase 8 — Brain Cloud Planning Foundation

Across Brain/Cloud/SDK, deliver capability discovery, cloud storage, APIs,
revisions, permissions, audit/events, and cloud-native brainstorm/spec foundations.

## Phase 9 — Hybrid Planning

Deliver explicit layer ownership, cloud-to-local artifacts and PRs, conflict
handling, sync metadata, offline behavior, recovery, and idempotency.

## Phase 10 — Unified Brain Cloud Planning UI

Deliver navigation, brainstorm, spec editor, roadmap, approvals, execution, Brain
context panel, and Hive integration inside the unified frontend.

## Phase 11 — Community Module Protocol

After official module contracts prove stable, deliver external-process protocol,
SDK direction, installation, permissions, compatibility/versioning, signing,
lifecycle, diagnostics, and distribution direction.

## Cross-Phase Gates

- Brain remains fully useful with Planning disabled.
- Every mutation has permission, confirmation, audit, and idempotency behavior.
- Every schema change has versioned migration and compatibility fixtures.
- Local work requires no hosted service.
- Planning never silently writes Brain memory.
- No phase claims security isolation not enforced by runtime.

## Exact Next Slice

> Review and merge the minimal internal module framework, then begin Phase 2
> Planning domain extraction from a separately approved spec. Do not import
> adapters, GitHub, Linear, Cobra, filesystem, or cloud dependencies into the
> Planning domain.
