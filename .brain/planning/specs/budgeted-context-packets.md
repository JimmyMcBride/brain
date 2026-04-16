---
created: "2026-04-16T04:36:00Z"
epic: budgeted-context-packets
project: brain
status: done
title: Budgeted Context Packets Spec
type: spec
updated: "2026-04-16T03:09:22Z"
---
# Budgeted Context Packets Spec

Created: 2026-04-16T04:36:00Z

## Why

Brain already emits summary-first packets, but it still mostly limits packet size by fixed counts such as “top 5 notes” or “top 4 boundaries.” That is a weak proxy for real token cost. Brain needs an explicit packet-budget model so it can optimize what fits into a useful context window instead of hoping the chosen counts stay small enough.

## Problem

The current compiler has no hard target for how large the final packet should be. Some packets stay compact by accident, others grow because the chosen summaries are individually verbose, and the system cannot explain what it omitted due to budget pressure because it does not yet have a budget concept.

## Goals

- Add explicit token-cost estimates for packet items and packet sections.
- Let `brain context compile` build under a deterministic budget target.
- Replace or supplement fixed item-count caps with budget-aware selection.
- Surface budget usage and omitted-item pressure in human and JSON output.
- Keep the first implementation simple, transparent, and local-first.

## Non-Goals

- Model-specific tokenizer parity.
- Dynamic budget tuning by remote model or provider.
- Aggressive semantic compression or LLM-generated rewriting at compile time.
- Replacing the existing ranking signals with a completely new scorer.

## Requirements

- Every `ContextItem` or compiled packet item must expose an estimated token cost.
- `brain context compile` must accept a budget target, either as named presets or an explicit integer token budget.
- Brain must reserve part of the budget for:
  - base contract
  - verification hints
  - ambiguities or diagnostics when present
- Working-set selection must operate under the remaining budget.
- The same repo state, task, and budget target must produce the same packet.
- Output must show:
  - target budget
  - estimated used budget
  - remaining budget
  - whether candidate items were omitted due to budget pressure
- The default budget should stay conservative enough to materially reduce startup packet size relative to the current compiler.
- First-wave preset and default targets should be justified against representative compile fixtures or sample tasks rather than hand-wavy constants alone.

## UX / Flows

Explicit budget:
1. User runs `brain context compile --task "tighten auth flow" --budget 1200 --json`.
2. Brain gathers the normal candidate set.
3. Brain estimates section and item costs, reserves budget for required sections, and selects the highest-value working-set items that fit.
4. Brain returns the packet plus budget diagnostics.

Preset budget:
1. User runs `brain context compile --task "tighten auth flow" --budget small`.
2. Brain resolves `small` to a built-in target.
3. Brain emits a leaner packet than the default and explains budget pressure when relevant.

## Data / Interfaces

Add packet-level diagnostics:
- `budget.target`
- `budget.used`
- `budget.remaining`
- `budget.reserve_base_contract`
- `budget.reserve_verification`
- `budget.omitted_due_to_budget`

Add item-level cost fields:
- `estimated_tokens`

Potential CLI surface:
- `brain context compile --budget small`
- `brain context compile --budget 1200`

## Risks / Open Questions

- Which sections should be mandatory even when the budget is extremely small?
- Should the compiler expose the top omitted candidates immediately, or only a count and pressure signal at first?
- Is word-count-based costing enough, or should Brain keep a slightly smarter heuristic for code-heavy summaries?

## Rollout

1. Add item-level cost accounting to compiler-facing context items and packet items.
2. Define packet budget presets and reserve rules.
3. Apply budget-aware selection to working-set candidates.
4. Confirm the default and preset budgets shrink representative packets without dropping mandatory sections.
5. Surface budget diagnostics in human and JSON output plus explain flows.
6. Update docs and the Brain skill after the interface settles.

## Story Breakdown

- [x] Add Estimated Token Cost Model And Budget Types
- [x] Add Compile Budget Inputs Presets And Stable Diagnostics
- [x] Make Working-Set Selection Budget-Aware And Deterministic
- [x] Surface Budget Pressure In Output Explain Docs And Brain Skill

## Resources

- [[.brain/brainstorms/token-efficient-context-direction.md]]
- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v1-context-packet-compiler.md]]
- [[.brain/planning/specs/v2-boundary-aware-context-selection.md]]
- [[.brain/planning/specs/v3-utility-aware-context-ranking.md]]

## Notes

This is the clearest 20/80 token-efficiency step. Brain already has transparent ranking inputs and summary-first packets; the missing piece is a real budget.

Implemented on `feature/context-packet-optimization` with deterministic token-cost heuristics, `small|default|large` presets plus explicit integer budgets, budget-aware working-set omission diagnostics in compile and explain output, and representative tests that prove tighter presets shrink the emitted working set while keeping mandatory sections.
