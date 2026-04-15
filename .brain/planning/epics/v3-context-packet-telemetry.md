---
created: "2026-04-15T03:55:57Z"
project: brain
spec: v3-context-packet-telemetry
title: V3 Context Packet Telemetry
type: epic
updated: "2026-04-15T04:01:06Z"
---
# V3 Context Packet Telemetry

Created: 2026-04-15T03:40:00Z

## Summary

Capture packet composition, expansions, and downstream outcome signals so Brain can evaluate context usefulness from real sessions instead of guessing from static design assumptions.

## Why It Matters

The compiler becomes much stronger once it can measure what actually helped. This epic creates the factual telemetry layer needed for later utility analysis and ranking improvements.

## Spec

- [[.brain/planning/specs/v3-context-packet-telemetry.md]]

## Sources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v1-packet-provenance-and-session-recording.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

This epic should capture packet facts and useful outcomes, not speculative reasoning. Keep the telemetry local, inspectable, and narrowly tied to compiler usefulness.
