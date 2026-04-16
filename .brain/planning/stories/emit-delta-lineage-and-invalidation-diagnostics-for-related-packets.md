---
created: "2026-04-16T02:07:44Z"
epic: session-packet-reuse
project: brain
spec: session-packet-reuse
status: done
title: Emit Delta Lineage And Invalidation Diagnostics For Related Packets
type: story
updated: "2026-04-16T04:39:24Z"
---
# Emit Delta Lineage And Invalidation Diagnostics For Related Packets

Created: 2026-04-16T02:07:44Z

## Description

When the task is still the same but compile inputs changed, emit concise delta lineage and invalidation diagnostics instead of pretending the new packet is unrelated.

## Acceptance Criteria

- [x] Same-task recompiles with a changed fingerprint return `cache_status=delta` plus lineage metadata that points to the previous packet.
- [x] The default delta response is compact and highlights only what changed instead of reprinting the whole packet unless Brain must fall back for standalone usability.
- [x] Delta diagnostics identify changed sections, changed item ids, or invalidation reasons without expanding into a second full packet dump.
- [x] Tests cover changed-file, touched-boundary, source-hash, and verification-requirement changes that should produce delta lineage rather than reuse.

## Resources

- [[.brain/planning/specs/session-packet-reuse.md]]

## Notes
