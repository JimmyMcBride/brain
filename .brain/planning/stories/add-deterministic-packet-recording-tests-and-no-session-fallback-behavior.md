---
created: "2026-04-15T05:32:39Z"
epic: v1-packet-provenance-and-session-recording
project: brain
spec: v1-packet-provenance-and-session-recording
status: done
title: Add Deterministic Packet Recording Tests And No-Session Fallback Behavior
type: story
updated: "2026-04-15T14:10:41Z"
---
# Add Deterministic Packet Recording Tests And No-Session Fallback Behavior

Created: 2026-04-15T05:32:39Z

## Description

Make packet recording safe and explainable by covering deterministic behavior, migration handling, and what happens when `context compile` runs outside an active session.


## Acceptance Criteria

- [ ] Tests prove packet recording is deterministic and inspectable for repeated compile runs
- [ ] The implementation has a clear and bounded behavior when `context compile` runs without an active session
- [ ] Packet recording changes do not regress existing session lifecycle or closeout behavior




## Resources

- [[.brain/planning/specs/v1-packet-provenance-and-session-recording.md]]

## Notes
