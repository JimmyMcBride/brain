---
created: "2026-04-16T04:37:00Z"
project: brain
spec: session-packet-reuse
title: Session Packet Reuse
type: epic
updated: "2026-04-16T04:39:24Z"
---
# Session Packet Reuse

Created: 2026-04-16T04:37:00Z

## Summary

Reuse compiled context packets inside active sessions when the task and relevant repo state are unchanged, and emit explicit packet deltas when the task remains stable but the working set shifts.

## Why It Matters

Always-injected rule systems save tokens at the start of a conversation but can become repeated prompt tax in long conversations. Brain can do better because it already knows the session, the task, and the recorded packet history. Reuse is only valuable if Brain can avoid re-emitting unchanged packet weight.

## Spec

- [[.brain/planning/specs/session-packet-reuse.md]]

## Sources

- [[.brain/brainstorms/token-efficient-context-direction.md]]
- [[.brain/planning/specs/v1-packet-provenance-and-session-recording.md]]
- [[.brain/planning/specs/v3-context-packet-telemetry.md]]
- [[.brain/planning/specs/v3-context-utility-analysis-surfaces.md]]

## Progress

- Spec completed.
- All four implementation stories are done.
- `brain context compile` now fingerprints relevant compile inputs, reuses the latest matching active-session packet as a compact response, emits compact `delta` responses with changed sections, changed item ids, and invalidation reasons when same-task inputs shift, and supports `--fresh` for explicit standalone full packets.
- `brain context explain --last`, `docs/usage.md`, and `skills/brain/SKILL.md` now surface cache status, reuse or delta lineage, invalidation reasons, and explicit full-packet fallback reasons.

## Notes

Keep the first reuse model session-local and conservative. The goal is to avoid repeated full packets when nothing relevant changed, not to build a cross-session cache platform.
