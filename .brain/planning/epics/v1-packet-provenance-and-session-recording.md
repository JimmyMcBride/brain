---
created: "2026-04-15T03:55:57Z"
project: brain
spec: v1-packet-provenance-and-session-recording
title: V1 Packet Provenance And Session Recording
type: epic
updated: "2026-04-15T03:59:02Z"
---
# V1 Packet Provenance And Session Recording

Created: 2026-04-15T03:40:00Z

## Summary

Record packet composition and provenance in session state so Brain can explain what it included now and build usefulness telemetry on top of real packet history later.

## Why It Matters

The compiler cannot be measured or improved if packet selection disappears after output. Recording packet composition creates the factual bridge between context selection, command execution, verification, and later tuning.

## Spec

- [[.brain/planning/specs/v1-packet-provenance-and-session-recording.md]]

## Sources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]
- [[.brain/planning/specs/live-work-context.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

Keep `v1` recording minimal and structural. This epic should capture packet facts, not invent a full telemetry or learning system early.
