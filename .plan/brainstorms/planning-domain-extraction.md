---
created_at: "2026-07-28T21:14:44Z"
project: brain
slug: planning-domain-extraction
status: active
title: Planning domain extraction
type: brainstorm
updated_at: "2026-07-28T21:16:51Z"
---

# Brainstorm: Planning domain extraction

Started: 2026-07-28T21:14:44Z

## Focus Question

Define the smallest storage-neutral Planning domain that preserves standalone Plan behavior needed by later local, GitHub, and hybrid adapters without importing Cobra, filesystem, GitHub, Linear, or cloud dependencies.
## Desired Outcome

One approved, implementation-ready spec for a small Planning domain package
that later local, GitHub, and hybrid adapters can share without importing their
storage or integration concerns.

## Vision

Planning has one deterministic core for its concepts and rules. Brain can use
that core as an optional official module, and standalone Plan can later share it
through compatibility wrappers. Specs remain the canonical execution contract;
runtime slices appear only when approved spec execution starts.

## Supporting Material

- `docs/planning/domain-model.md`
- `docs/planning/plan-feature-inventory.md`
- `docs/planning/implementation-roadmap.md`
- `docs/planning/storage-and-modes.md`
- `docs/modules/planning-driven-requirements.md`
- `JimmyMcBride/plan` `develop`, especially `internal/planning` and its tests

## Constraints

- No Cobra, filesystem, GitHub, Linear, cloud, module-runtime, or Brain private-storage dependencies in the domain package.
- Brain Core must remain useful with Planning disabled.
- Specs remain canonical; execution slices are ephemeral runtime output, not pre-created durable artifacts.
- Support only local, github, and hybrid ownership concepts; do not represent Linear.
- Preserve deterministic behavior and compatibility semantics proven by standalone Plan tests.
- Do not build adapters, CLI commands, persistence, migrations, or module enablement in Phase 2.

## Open Questions

- Which existing Plan behaviors are truly domain rules versus Markdown/workspace application behavior that belongs in Phase 3?
- What minimum artifact representation preserves identity, status, dependencies, provenance, and deterministic execution without embedding storage formats?
- Should guided-session state and promotion reconciliation enter Phase 2, or remain application workflow until a second adapter proves the abstraction?
## Ideas

- Model brainstorms, specs, initiatives, roadmaps, readiness, approval, execution queues, and runtime slices as storage-neutral domain concepts.
- Preserve specs as the canonical execution contract; create slices only at execution runtime and retain traceability through commits and the final PR.
- Define ports and deterministic domain services only where current standalone Plan behavior proves they are needed; keep all persistence and integration concerns outside the domain.
- Exclude Linear code, types, schemas, permissions, commands, and migration behavior from Brain Planning.
## Raw Notes

## Refinement

### Problem

Planning behavior in standalone Plan is coupled to CLI, workspace files, and integration-specific types, so moving it directly into Brain would reproduce those dependencies and make later local, GitHub, and hybrid adapters hard to share or test.

### User / Value

Brain users gain an optional native Planning capability that stays local-first and integration-agnostic; standalone Plan can later share the same behavior through compatibility wrappers; maintainers get deterministic domain tests and clear adapter boundaries.

### Appetite

One reviewable spec and implementation PR that extracts only the domain model and pure services needed to support later local-module work, with focused unit tests and no end-user command surface.

### Remaining Open Questions

- Which existing Plan behaviors are truly domain rules versus Markdown/workspace application behavior that belongs in Phase 3?
- What minimum artifact representation preserves identity, status, dependencies, provenance, and deterministic execution without embedding storage formats?
- Should guided-session state and promotion reconciliation enter Phase 2, or remain application workflow until a second adapter proves the abstraction?

### Candidate Approaches

- Extract a narrow pure-domain kernel: artifact/value types, validation and state transitions, readiness/dependency evaluation, and deterministic runtime-slice derivation; leave repositories, parsing, guided UI, and publication for later phases.
- Extract most of internal/planning behind generic repositories in one pass; broader compatibility but much larger coupling and review risk.
- Keep Plan unchanged and duplicate a fresh Brain domain; fastest locally but creates drift and weakens the compatibility path.

### Decision Snapshot

Proceed with one bounded spec for a narrow pure-domain kernel. Include validated
artifact identities and states, explicit spec/approval transitions,
readiness/dependency evaluation, execution queues, and deterministic runtime
slice derivation. Keep repositories, Markdown parsing, guided-session
orchestration, module wiring, CLI behavior, persistence, migrations, GitHub,
Linear, and cloud work outside Phase 2.

## Challenge

### Rabbit Holes

- Treating every existing internal/planning function as domain behavior.
- Designing generic repository or event interfaces before two implementations need them.
- Recreating Markdown parsing, workspace schemas, CLI JSON, or guided prompts inside the domain.
- Solving Phase 3 local persistence or Phase 5 GitHub reconciliation early.
- Designing cloud sync or community-module protocols.

### No-Gos

- No Linear symbols or compatibility logic.
- No filesystem, Cobra, GitHub API, cloud SDK, or Brain storage imports.
- No durable slice files and no slice creation before spec execution begins.
- No user-facing brain plan commands, module registration, or migration writes.
- No speculative extension framework beyond concrete Phase 2 needs.

### Assumptions

- Standalone Plan tests are the compatibility evidence, but package layout and storage representations are not contracts.
- A narrow shared domain can be introduced in Brain first and consumed by standalone Plan during Phase 4 without immediate cross-repository packaging.
- Artifact content can remain opaque or structured only where a domain rule needs it; adapters will own Markdown serialization.
- Pure functions and explicit state transitions are sufficient for Phase 2.

### Likely Overengineering

A universal artifact graph, generic repository framework, pluggable rule engine, event sourcing, or full workflow state machine would exceed the evidence and postpone the local adapter that should validate the design.

### Simpler Alternative

Create one internal/planning domain package containing validated identifiers and statuses, compact artifact metadata/value types, allowed spec transitions, dependency/readiness evaluation, execution-plan derivation, and runtime-slice types. Prove each behavior with table-driven tests and leave all I/O and orchestration for later phases.
