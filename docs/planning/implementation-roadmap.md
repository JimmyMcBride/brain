---
updated: "2026-09-01T14:26:27Z"
---
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
`.plan/specs/minimal-internal-module-framework.md` and merged by PR #37.

Deliver module descriptor/interface, registry, capability declarations,
project-scoped enable/disable state, namespaced configuration, permissions,
minimal lifecycle, health, tests, and one test-only reference module.

Do not load external community processes. Do not migrate Planning domain code.

## Phase 2 — Planning Domain Extraction

Status: Implemented by `.plan/specs/planning-domain-extraction.md` and merged
by PR #38.

Deliver storage-neutral brainstorm, spec, initiative, roadmap, readiness,
approval, execution, queue, and runtime-slice models with unit tests. Keep Cobra,
filesystem, GitHub, and cloud dependencies outside domain packages. Do not
extract Linear code or types.

## Phase 3 — Local Planning Module

Status: Implemented from `.plan/specs/local-planning-module.md` and merged by
PR #39.

Deliver module enablement, native command group, `.plan/` read/write compatibility,
workspace detection/migration, Brain context access, Planning events/permissions,
and local tests.

## Phase 4 — Plan CLI Compatibility

Status: In progress from `.plan/specs/plan-cli-compatibility.md`. PR #46
pinned standalone Plan revision `53ebd96`, recorded the complete command
disposition, and captured initial black-box `status` cases. PR #47 normalized
Brain's Go module path. PR #48 exposed the stdlib-only domain at
`github.com/JimmyMcBride/brain/planning`. The fourth slice captures status,
project/spec check, and roadmap contracts; exposes importable
`planning/application` and `planning/local` packages with locked exports and
dependencies; and migrates native aggregate status, check, and guarded roadmap
commands behind shared conformance tests. The fifth slice captures and migrates
local idea/refinement/challenge updates, guided-session navigation and packets,
roadmap parking, maturity/repair behavior, and confirmed direct promotion from
brainstorm to canonical spec. Content and promotion mutations are preview-first;
all writes are permissioned, confirmed, audited, atomic at the local adapter
boundary, and idempotent. Direct promotion creates no epic or persisted
execution-slice intermediate.
The sixth slice captures standalone spec edit, lifecycle, analysis, checklist,
initiative, execution, and handoff contracts, then migrates them behind shared
application/local seams. Brain previews mutations by default, requires explicit
spec edit/approval permission and confirmation, emits audit events, derives
runtime slices without story files, and compensates an execution-status write
when guided-session handoff persistence fails.
The seventh slice merged through standalone Plan PR
[`#87`](https://github.com/JimmyMcBride/plan/pull/87). It pins Brain merge
`c2c71279030f` without a local replacement, routes compatible schema-v3 local
families through the shared packages, preserves standalone host and fallback
behavior, and enables the interactive-only warning policy. Linux and Windows CI
are green. Final rollout replaces the pseudo-version with the next published
Brain tag before publishing the corresponding standalone Plan release.

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

> Publish the merged Brain Planning boundary as the next stable Brain tag,
> replace standalone Plan's temporary pseudo-version with that tag, repeat the
> cross-repository verification matrix, and publish the corresponding Plan
> release. Then mark Phase 4 complete and advance to Phase 5.
