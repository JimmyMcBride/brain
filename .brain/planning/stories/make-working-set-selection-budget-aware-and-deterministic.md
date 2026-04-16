---
created: "2026-04-16T02:07:44Z"
epic: budgeted-context-packets
project: brain
spec: budgeted-context-packets
status: done
title: Make Working-Set Selection Budget-Aware And Deterministic
type: story
updated: "2026-04-16T03:09:22Z"
---
# Make Working-Set Selection Budget-Aware And Deterministic

Created: 2026-04-16T02:07:44Z

## Description

Apply hard budget pressure to working-set selection so Brain keeps the highest-value items that fit, preserves mandatory sections, and omits lower-priority candidates in a reproducible order.

## Acceptance Criteria

- [ ] Working-set selection honors reserve budgets before choosing optional candidates.
- [ ] When candidates exceed the remaining budget, Brain omits lower-priority items deterministically for the same repo state, task, and budget target.
- [ ] Mandatory packet sections still appear under tight budgets, with tests covering small-budget edge cases.

## Resources

- [[.brain/planning/specs/budgeted-context-packets.md]]

## Notes
