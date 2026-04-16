---
created: "2026-04-16T02:07:44Z"
epic: budgeted-context-packets
project: brain
spec: budgeted-context-packets
status: done
title: Add Compile Budget Inputs Presets And Stable Diagnostics
type: story
updated: "2026-04-16T03:09:22Z"
---
# Add Compile Budget Inputs Presets And Stable Diagnostics

Created: 2026-04-16T02:07:44Z

## Description

Expose packet-budget control on `brain context compile` and lock the budget diagnostics schema so users can ask for smaller or larger packets without guessing what the compiler did.

## Acceptance Criteria

- [ ] `brain context compile` accepts `--budget <preset|integer>` with clear validation for unknown presets and invalid numeric values.
- [ ] Default, small, and any other first-wave presets resolve to deterministic target budgets that materially cut startup packet size versus the current default behavior.
- [ ] Preset defaults are justified with representative before/after packet fixtures or sample tasks instead of arbitrary constants alone.
- [ ] Human and JSON output expose target, used, remaining, and reserve fields in one stable diagnostics shape.

## Resources

- [[.brain/planning/specs/budgeted-context-packets.md]]

## Notes
