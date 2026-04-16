---
created: "2026-04-16T02:07:44Z"
epic: derived-doc-capsules-and-drift-audit
project: brain
spec: derived-doc-capsules-and-drift-audit
status: todo
title: Prefer Capsules During Compile And Expand Canonical Docs Only When Needed
type: story
updated: "2026-04-16T05:39:00Z"
---
# Prefer Capsules During Compile And Expand Canonical Docs Only When Needed

Created: 2026-04-16T02:07:44Z

## Description

Teach the compiler to pick capsule forms first for eligible docs, then fall back to full canonical docs only when ambiguity, explicit intent, or capsule invalidity demands it.

## Acceptance Criteria

- [ ] The compiler supports `capsules=off|auto|on`, with `off` leaving normal summaries in place, `on` forcing eligible capsule use for evaluation, and `auto` deciding per compile from local fresh-packet telemetry.
- [ ] In `auto`, Brain uses only local fresh-packet telemetry plus current capsule health to decide whether eligible docs should go through capsules for that compile; reused and delta packets do not count toward the trigger.
- [ ] For first-wave source docs, compiled packets prefer capsule items only when the current mode and telemetry justify them, while preserving anchors back to the full canonical source.
- [ ] When ambiguity, explicit user intent, or invalid capsule state requires more detail, Brain expands to the full source doc instead of forcing the capsule path.
- [ ] Compile or explain diagnostics make the capsule decision visible, including whether capsules were off, auto-selected, or forced on, and the reason.
- [ ] Compiler tests cover mixed packets where some docs stay capsule-backed and others expand to canonical source content.

## Resources

- [[.brain/planning/specs/derived-doc-capsules-and-drift-audit.md]]

## Notes
