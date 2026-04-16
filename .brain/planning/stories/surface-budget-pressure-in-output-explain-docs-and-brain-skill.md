---
created: "2026-04-16T02:07:44Z"
epic: budgeted-context-packets
project: brain
spec: budgeted-context-packets
status: done
title: Surface Budget Pressure In Output Explain Docs And Brain Skill
type: story
updated: "2026-04-16T03:09:22Z"
---
# Surface Budget Pressure In Output Explain Docs And Brain Skill

Created: 2026-04-16T02:07:44Z

## Description

Make budget pressure visible everywhere users inspect packets, then teach the new budget model in docs and the Brain skill so the feature stays understandable and debuggable.

## Acceptance Criteria

- [ ] Packet output and explain surfaces identify when items were omitted because of budget pressure and expose at least a count or top omitted candidates.
- [ ] `docs/usage.md` and `skills/brain/SKILL.md` teach budgeted compile, preset versus explicit budgets, and how to inspect budget diagnostics.
- [ ] User guidance keeps the first wave framed as deterministic local heuristics rather than a model-specific token estimator.

## Resources

- [[.brain/planning/specs/budgeted-context-packets.md]]
- [[docs/usage.md]]
- [[skills/brain/SKILL.md]]

## Notes
