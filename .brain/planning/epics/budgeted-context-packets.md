---
created: "2026-04-16T04:36:00Z"
project: brain
spec: budgeted-context-packets
title: Budgeted Context Packets
type: epic
updated: "2026-04-16T02:41:00Z"
---
# Budgeted Context Packets

Created: 2026-04-16T04:36:00Z

## Summary

Add explicit token-cost accounting and packet budgets to `brain context compile` so Brain selects the smallest justified working set under a hard target instead of relying mostly on fixed item-count caps.

## Why It Matters

Brain already compiles summary-first packets, but it does not yet optimize against a true packet budget. This means packet size is still only indirectly controlled. Budget-aware selection is the fastest way to cut startup context size without giving up boundary-aware quality.

## Spec

- [[.brain/planning/specs/budgeted-context-packets.md]]

## Sources

- [[.brain/brainstorms/token-efficient-context-direction.md]]
- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v1-context-packet-compiler.md]]
- [[.brain/planning/specs/v2-boundary-aware-context-selection.md]]
- [[.brain/planning/specs/v3-utility-aware-context-ranking.md]]

## Progress

- Spec completed.
- All four implementation stories are done.
- `brain context compile` now supports deterministic packet budgets with token estimates, presets, budget-aware working-set omission, and budget diagnostics in compile plus explain surfaces.
- Representative tests now lock the real value claim: tighter presets emit leaner working sets while keeping mandatory sections.

## Notes

Keep this deterministic. The first budgeted compiler should use transparent cost heuristics and explicit reserve budgets for base contract and verification hints rather than chasing tokenizer-perfect precision.
