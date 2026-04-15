---
created: "2026-04-15T05:32:39Z"
epic: v1-packet-provenance-and-session-recording
project: brain
spec: v1-packet-provenance-and-session-recording
status: todo
title: Extend Session State For Packet Records
type: story
updated: "2026-04-15T05:32:39Z"
---
# Extend Session State For Packet Records

Created: 2026-04-15T05:32:39Z

## Description

Add the minimum session-state structures needed to record compiled packet identity, included item IDs, anchors, and inclusion reasons without storing expanded source bodies or transcript-like scratch.


## Acceptance Criteria

- [ ] Session state can persist packet identity, task summary, included item IDs, anchors, and inclusion reasons
- [ ] The stored representation avoids full source copies and remains inspectable and migration-friendly
- [ ] Packet recording structures can support repeated compile events within a single session without destructive overwrites




## Resources

- [[.brain/planning/specs/v1-packet-provenance-and-session-recording.md]]

## Notes
