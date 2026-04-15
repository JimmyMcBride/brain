---
created: "2026-04-15T05:33:26Z"
epic: v2-boundary-aware-context-selection
project: brain
spec: v2-boundary-aware-context-selection
status: done
title: Balance Multi-Boundary Packets And Surface Boundary Diagnostics
type: story
updated: "2026-04-15T12:00:00Z"
---
# Balance Multi-Boundary Packets And Surface Boundary Diagnostics

Created: 2026-04-15T05:33:26Z

## Description

Keep boundary-aware selection sharp for tasks spanning multiple areas of the repo by balancing packet allocation across boundaries and surfacing diagnostics that explain why a boundary was emphasized or suppressed.


## Acceptance Criteria

- [ ] The compiler can assemble bounded packets for multi-boundary tasks without flooding all neighboring areas
- [ ] Boundary-aware diagnostics explain inclusion because of changed files, direct overlap, or adjacency without becoming opaque ranking jargon
- [ ] Packet balancing reduces noisy neighbors in large or dense directories while preserving the minimum justified coverage for each touched boundary




## Resources

- [[.brain/planning/specs/v2-boundary-aware-context-selection.md]]


## Notes
