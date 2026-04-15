---
created: "2026-04-15T05:32:39Z"
epic: v1-context-packet-compiler
project: brain
spec: v1-context-packet-compiler
status: done
title: Implement First-Wave Candidate Gathering And Deterministic Selection
type: story
updated: "2026-04-15T14:10:41Z"
---
# Implement First-Wave Candidate Gathering And Deterministic Selection

Created: 2026-04-15T05:32:39Z

## Description

Build the first-wave compiler selection path from base-contract items, changed files, touched boundaries, nearby tests, and durable note summaries using deterministic rather than adaptive ranking.


## Acceptance Criteria

- [ ] The first-wave compiler gathers candidates from the approved base-contract, live-work, and durable-note channels
- [ ] Selection stays deterministic for the same repo state and task input and remains bounded enough for practical agent startup
- [ ] The initial selector avoids telemetry-aware reranking and does not require a standalone context index to function




## Resources

- [[.brain/planning/specs/v1-context-packet-compiler.md]]

## Notes
