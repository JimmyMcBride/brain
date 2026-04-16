---
created: "2026-04-16T02:07:44Z"
epic: session-packet-reuse
project: brain
spec: session-packet-reuse
status: done
title: Add Compile Fingerprint And Reuse Lineage To Session Packets
type: story
updated: "2026-04-16T04:39:24Z"
---
# Add Compile Fingerprint And Reuse Lineage To Session Packets

Created: 2026-04-16T02:07:44Z

## Description

Define the compile fingerprint and lineage metadata Brain needs to decide whether a packet can be reused or should only be related by delta inside the active session.

## Acceptance Criteria

- [x] Recorded session packets store a deterministic fingerprint built from the compile inputs that materially affect packet validity.
- [x] Packet metadata includes stable reuse and delta lineage fields without weakening existing provenance fields.
- [x] Tests cover fingerprint stability for unchanged inputs and invalidation when relevant task, repo-state, or policy inputs change.

## Resources

- [[.brain/planning/specs/session-packet-reuse.md]]

## Notes
