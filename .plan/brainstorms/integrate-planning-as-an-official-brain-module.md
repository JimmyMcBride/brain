---
created_at: "2026-07-27T21:38:28Z"
project: brain
slug: integrate-planning-as-an-official-brain-module
status: active
title: Integrate Planning as an official Brain module
type: brainstorm
updated_at: "2026-07-27T21:40:11Z"
---

# Brainstorm: Integrate Planning as an official Brain module

Started: 2026-07-27T21:38:28Z

## Focus Question

## Desired Outcome

Define reviewed module contracts and a phased migration path that makes Planning optional but native, while Brain Core remains useful without it.
## Vision

Brain becomes the shared platform for project identity, context, memory,
retrieval, sessions, permissions, events, audit, and module lifecycle. Planning
becomes an official module users can enable when they want Brain-native shaping,
specification, approval, and execution guidance. Existing Plan users keep a safe
compatibility path, and future official/community modules use the same public
extension foundations.

## Supporting Material

- `docs/modules/overview.md`
- `docs/modules/architecture.md`
- `docs/modules/planning-driven-requirements.md`
- `docs/planning/plan-feature-inventory.md`
- `docs/planning/implementation-roadmap.md`
- `https://github.com/JimmyMcBride/plan` at inspected `develop` commit `53ebd96`

## Constraints

- Brain Core stays useful with Planning disabled.
- No major Planning port before the module framework is reviewed.
- Retain .plan and Plan CLI compatibility during migration.
- No official Linear commitment; GitHub remains transitional.
- Do not claim external-process sandboxing or choose transport yet.

## Open Questions

- What minimal persisted module enablement/config shape should Phase 1 use?
- Which Plan JSON contracts have real external consumers and need a compatibility window?
- What Cloud Planning capability is the exit condition for canonical GitHub source mode?
## Ideas

- Preserve brainstorms, refinement, challenge, maturity, specs, initiatives, roadmaps, readiness, preview/confirmation, idempotent reconciliation, queues, and runtime slices.

- Redesign identity, configuration, permissions, events, audit, context/retrieval access, cloud storage, tools, health, and memory proposals through Brain Core contracts.

- Keep GitHub transitional behind future generic Planning adapters; retire official Linear with explicit old-workspace diagnostics and migration/export.

- Use local, cloud, and explicit layer-owned hybrid Planning modes; treat Planning sync separately from Brain context-memory sync.
## Raw Notes

## Refinement

### Problem

Brain and Plan have complementary foundations but separate product identities, storage, CLI, cloud direction, and extension assumptions. A direct merge would duplicate platform systems or force Planning on every Brain user.

### User / Value

Brain users can opt into native planning; current Plan users retain artifacts and scripts during migration; teams and community authors gain stable module contracts without adopting one mandated workflow.

### Appetite

Phase 0 is documentation, inventory, ADRs, and migration contracts only. First implementation slice is a minimal internal module framework with one trivial reference module.

### Remaining Open Questions

- What minimal persisted module enablement/config shape should Phase 1 use?
- Which Plan JSON contracts have real external consumers and need a compatibility window?
- What Cloud Planning capability is the exit condition for canonical GitHub source mode?

### Candidate Approaches

- Build minimal compiled-module registry/runtime first, then extract storage-neutral Planning domain.
- Retain .plan locally and share implementation between brain plan and compatibility plan.
- Move GitHub behind adapters after local module works; remove Linear only after diagnostics/export.
- Add Brain knowledge loop and Cloud/hybrid capabilities in later gated phases.

### Decision Snapshot

## Challenge

### Rabbit Holes

- Porting internal/planning line-for-line before package boundaries are reviewed.
- Designing external-process transport, marketplace, signing, and frontend extension systems in Phase 0.
- Reorganizing all existing Brain packages into an imagined core tree.
- Preserving every GitHub or Linear feature in falsely generic interfaces.

### No-Gos

- No major migration code in architecture phase.
- No silent Brain memory writes.
- No Planning requirement for Brain Core users.
- No immediate Plan CLI break or .plan relocation.
- No claim that community modules are sandboxed.

### Assumptions

- Current .plan artifacts and CLI semantics can remain readable while shared services are extracted.
- Brain Core facades can expose needed capabilities without leaking concrete managers or SQLite.
- A compiled reference module will reveal enough lifecycle issues before Planning moves.
- Brain Cloud can advertise Planning as an optional capability.

### Likely Overengineering

Freezing a comprehensive public manifest and protocol before one trivial compiled module proves registration, enablement, configuration, permissions, lifecycle, and health.

### Simpler Alternative

Implement only an internal compiled-module registry with explicit project enablement, namespaced configuration, permissions, lifecycle hooks, health, tests, and one no-op reference module. Review it before extracting any Planning domain code.
