---
created: "2026-04-16T04:38:00Z"
epic: derived-doc-capsules-and-drift-audit
project: brain
status: approved
title: Derived Doc Capsules And Drift Audit Spec
type: spec
updated: "2026-04-16T05:39:00Z"
---
# Derived Doc Capsules And Drift Audit Spec

Created: 2026-04-16T04:38:00Z

## Why

Large Brain-managed docs are still expensive to pull repeatedly, and the current summary layer is partly hand-shaped and narrow. Brain needs a more general way to derive compact capsule forms from source docs, then verify those capsules are still aligned when the source docs change.

## Problem

Today Brain has summaries, but not a generalized capsule system. That means:
- summary coverage is uneven
- source-doc drift can go unnoticed
- the compiler cannot reliably distinguish between a trustworthy capsule and a stale one
- the current planning direction still risks overbuilding if budgets and session reuse solve most of the remaining packet pressure

## Goals

- Introduce a first-class derived capsule model for selected Brain-managed docs.
- Keep full docs canonical and human-readable.
- Make capsules small, compiler-friendly, and anchored to exact source sections.
- Add drift auditing so stale capsules are visible and fixable.
- Support explicit capsule usage modes of `off`, `auto`, and `on` so the feature stays predictable and debuggable.
- When capsules are enabled and justified, prefer capsule forms in compiled packets and only expand full sources when needed.
- Build on the existing summary-and-anchor path instead of creating a parallel summary stack.

## Non-Goals

- A generic editor rule-engine or always-injected `.mdc` system.
- Semantic proof that a capsule is perfectly equivalent to the full doc.
- Making every repo doc participate in the first wave.
- Hiding source-doc drift behind silent auto-rewrites.
- Creating a second independent summary abstraction when the existing `ContextItem` and anchor pipeline can be extended.
- A hidden permanent self-enabling switch that silently changes compile behavior without an explicit, inspectable reason.

## Requirements

- Brain must define a `capsule` or equivalent derived-item model with:
  - source path
  - source section or anchor
  - source hash
  - capsule content
  - estimated token cost
- The capsule model should reuse or cleanly specialize the existing summary-and-anchor pipeline wherever possible.
- First-wave capsules should target only the most compiler-relevant Brain-managed docs that still dominate packet cost after budgeted compile and session reuse have landed, such as:
  - `AGENTS.md`
  - `.brain/context/workflows.md`
  - `.brain/context/memory-policy.md`
  - `.brain/context/architecture.md`
  - selected generated project docs when useful
- Capsule generation must be deterministic.
- Brain must detect stale or missing capsules when the source hash changed.
- Brain must surface capsule drift through an inspectable path such as `brain doctor`, a context-specific audit command, or both.
- Brain must support `capsules=off|auto|on` or an equivalent operator-facing control surface.
- The first shipping state for capsules should default to `off`.
- In `auto`, the compiler must evaluate local fresh-packet telemetry and current capsule health per compile, then use capsules only when conservative evidence shows they are likely to help.
- Auto decisions must use fresh-packet telemetry only, not reused or delta packets.
- Auto decisions must be per compile and explainable, not a one-way persistent self-mutation of repo behavior.
- The compiler must prefer capsule forms only when the current mode and capsule health allow it, and must still expand the full source when ambiguity or explicit user intent demands it.
- The first-wave audit should support stricter checklist-style validation for docs that opt into explicit capsule coverage metadata.
- If budgets plus reuse remove most of the remaining doc-cost pressure, Brain should be able to stop with a minimal capsule wave rather than force broad coverage.
- `context explain` or an equivalent inspectable surface must say whether capsules were off, auto-selected, or forced on, and why.

## UX / Flows

Capsule-backed compile:
1. User runs `brain context compile --task "tighten auth flow"` after budgets and session reuse already exist.
2. Brain selects capsule items for the few relevant docs that still carry meaningful packet cost instead of heavier full-doc summaries.
3. Brain attaches exact anchors back to the full source.

Telemetry-gated auto compile:
1. User or project config enables capsule mode `auto`.
2. Brain inspects local fresh-packet telemetry for recurring budget pressure and recurring omission of capsule-eligible docs.
3. If the thresholds are met and the relevant capsules are current, Brain uses capsule items for that compile.
4. If the thresholds are not met or the capsules are stale, Brain stays on the normal summary path.
5. `brain context explain` reports whether capsules were off, auto-selected, or forced on, plus the decision reason.

Drift audit:
1. A workflow or doc changes materially.
2. Capsule drift is detected because the source hash no longer matches the derived capsule state.
3. Brain surfaces the stale capsule and points to the source doc that needs regeneration or review.

## Data / Interfaces

Potential capsule fields:
- `capsule_id`
- `source_path`
- `source_section`
- `source_hash`
- `content`
- `estimated_tokens`
- `coverage_mode`
- `coverage_count`

Potential audit output:
- `current`
- `missing`
- `stale`
- `coverage_mismatch`

Potential surfaces:
- `brain doctor`
- `brain context capsules audit`
- compiler diagnostics when a stale capsule would otherwise be used

Potential mode and auto-gating inputs:
- `capsules=off|auto|on`
- fresh packets analyzed
- fresh packet pressure rate
- recurring omitted eligible docs
- distinct tasks represented in the omission sample
- capsule drift status
- explainable selection reason when `auto` turns capsules on for a compile

## Risks / Open Questions

- Which docs are structured enough for strict checklist-count auditing, and which only support hash-based drift detection?
- Should capsules be stored as explicit files, derived state in `.brain/state`, or rebuilt on demand?
- How many capsule layers are actually useful before the system becomes harder to reason about?
- What evidence threshold is enough to justify starting this third wave after budgets and reuse ship?
- What initial conservative auto thresholds are strong enough to avoid over-triggering on a small repo while still surfacing legitimate first-turn packet pain on larger ones?

## Rollout

1. Reassess remaining document-cost pressure after budgeted compile and session reuse land.
2. If this epic is revived, add explicit `off|auto|on` mode control and explainable auto-gating before broad capsule coverage.
3. Define the capsule data model and select a minimal first-wave source set.
4. Generate deterministic capsules for those docs by extending the existing summary-and-anchor path.
5. Teach the compiler to prefer capsules over heavier source summaries only when the current mode and telemetry justify it.
6. Add drift auditing and visible status reporting.
7. Update docs and the Brain skill to explain capsules as derived, auditable inputs.
8. Expand coverage only after the first-wave docs prove useful.

## Story Breakdown

- [ ] Define Derived Capsule Model And First-Wave Source Coverage
- [ ] Build Deterministic Capsule Generation And Persistence
- [ ] Prefer Capsules During Compile And Expand Canonical Docs Only When Needed
- [ ] Add Capsule Drift Audit And Doctor Diagnostics
- [ ] Teach Docs And Brain Skill About Capsules And Drift Audit

## Resources

- [[.brain/brainstorms/token-efficient-context-direction.md]]
- [[.brain/planning/specs/v1-base-contract-and-summary-anchors.md]]
- [[.brain/planning/specs/v1-context-packet-compiler.md]]
- [[.brain/planning/specs/release-install-and-update-flow.md]]

## Notes

This is how Brain should borrow the useful part of condensed rule files: derived compact context with explicit auditability, not permanent prompt reinjection.

Re-evaluated on 2026-04-16 after budgets plus session reuse landed: representative repeated compiles already fell to roughly `281-287` human-estimated tokens and `412-421` JSON-estimated tokens from fresh responses near `1425-1431` human-estimated tokens and `2781-2787` JSON-estimated tokens. Capsules remain third-wave and should stay deferred until there is concrete evidence that first-turn packet cost, not repeated packet weight, is still limiting Brain enough to justify the added drift-audit surface.

Refined on 2026-04-16: if capsules ever come back, they should return as an explicitly constrained feature with `off|auto|on` mode control, default `off`, conservative telemetry-gated `auto`, and per-compile explainability rather than as a silent permanent self-enable path.
