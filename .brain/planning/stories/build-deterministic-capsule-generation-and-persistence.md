---
created: "2026-04-16T02:07:44Z"
epic: derived-doc-capsules-and-drift-audit
project: brain
spec: derived-doc-capsules-and-drift-audit
status: todo
title: Build Deterministic Capsule Generation And Persistence
type: story
updated: "2026-04-16T02:41:00Z"
---
# Build Deterministic Capsule Generation And Persistence

Created: 2026-04-16T02:07:44Z

## Description

Implement deterministic generation and inspectable persistence for first-wave capsules so Brain can rebuild them repeatably and compare them against source docs.

## Acceptance Criteria

- [ ] Capsule generation is deterministic for the same source content and emits exact source anchors plus estimated token costs.
- [ ] Generation reuses existing summary-and-anchor derivation primitives where practical instead of standing up a disconnected pipeline.
- [ ] Generated capsules live in one inspectable derived location with tests for rebuild stability and unchanged reruns.
- [ ] The persistence path does not create a generic free-form rule-pack system or hide how capsules were derived.

## Resources

- [[.brain/planning/specs/derived-doc-capsules-and-drift-audit.md]]

## Notes
