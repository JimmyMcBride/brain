---
created: "2026-04-16T02:07:44Z"
epic: budgeted-context-packets
project: brain
spec: budgeted-context-packets
status: done
title: Add Estimated Token Cost Model And Budget Types
type: story
updated: "2026-04-16T03:09:22Z"
---
# Add Estimated Token Cost Model And Budget Types

Created: 2026-04-16T02:07:44Z

## Description

Add deterministic token-cost heuristics and packet-budget types so the compiler can reason about packet size with transparent local estimates instead of item-count proxies alone.

## Acceptance Criteria

- [ ] Compiler-facing context items and compiled packet items expose estimated token cost fields that can be aggregated by section.
- [ ] Budget configuration supports named presets plus explicit integer targets with reserve buckets for base contract, verification, and diagnostics.
- [ ] Cost estimation stays deterministic and local-first, with tests that avoid promising tokenizer-perfect parity.

## Resources

- [[.brain/planning/specs/budgeted-context-packets.md]]

## Notes
