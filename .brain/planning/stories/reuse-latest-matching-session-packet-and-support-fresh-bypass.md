---
created: "2026-04-16T02:07:44Z"
epic: session-packet-reuse
project: brain
spec: session-packet-reuse
status: done
title: Reuse Latest Matching Session Packet And Support Fresh Bypass
type: story
updated: "2026-04-16T04:39:24Z"
---
# Reuse Latest Matching Session Packet And Support Fresh Bypass

Created: 2026-04-16T02:07:44Z

## Description

Teach `brain context compile` to short-circuit to the latest matching packet inside the active session while still allowing users to force a fresh compile for debugging.

## Acceptance Criteria

- [x] When the latest active-session packet fingerprint matches, `brain context compile` returns that packet as reused instead of rebuilding it.
- [x] The default reused response is materially smaller than a fresh full packet and does not re-emit unchanged packet sections wholesale.
- [x] `brain context compile --fresh` bypasses reuse even when the fingerprint matches and records the new packet as a fresh compile.
- [x] When no active session exists or no packet matches, compile falls back to normal packet generation without hidden reuse behavior.

## Resources

- [[.brain/planning/specs/session-packet-reuse.md]]

## Notes
