---
created: "2026-04-16T02:07:44Z"
epic: session-packet-reuse
project: brain
spec: session-packet-reuse
status: done
title: Teach Explain Docs And Brain Skill About Packet Reuse
type: story
updated: "2026-04-16T04:39:24Z"
---
# Teach Explain Docs And Brain Skill About Packet Reuse

Created: 2026-04-16T02:07:44Z

## Description

Surface packet reuse and delta behavior in inspectable user-facing flows, then update docs and the Brain skill so the feature is visible and easy to override.

## Acceptance Criteria

- [x] `brain context explain --last` shows reuse lineage, delta lineage, invalidation reasons, and whether Brain emitted a compact reuse or delta response versus a full fallback packet.
- [x] `docs/usage.md` and `skills/brain/SKILL.md` explain automatic session-local reuse, compact delta behavior, and the `--fresh` escape hatch.
- [x] The guidance makes clear that reuse is conservative and session-local, not a hidden cross-session cache platform.

## Resources

- [[.brain/planning/specs/session-packet-reuse.md]]
- [[docs/usage.md]]
- [[skills/brain/SKILL.md]]

## Notes
