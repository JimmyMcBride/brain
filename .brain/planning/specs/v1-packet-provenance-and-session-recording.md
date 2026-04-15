---
created: "2026-04-15T03:55:57Z"
epic: v1-packet-provenance-and-session-recording
project: brain
status: approved
title: V1 Packet Provenance And Session Recording Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V1 Packet Provenance And Session Recording Spec

Created: 2026-04-15T03:40:00Z

## Why

The context compiler needs a durable trail of what it selected for a task. Without that trail, Brain cannot later connect packet contents to verification outcomes, durable updates, or utility signals.

## Problem

Current session state records commands and workflow enforcement, but not the context packet that shaped the work. That leaves Brain unable to answer basic questions such as which context items were included, which packet shape was used, or whether the same task repeatedly compiles the same working set.

## Goals

- Record packet composition for compiled context packets.
- Preserve inclusion reasons and packet identity in session state.
- Keep the first-wave recording light enough to ship with `v1`.
- Prepare cleanly for later telemetry and utility scoring.

## Non-Goals

- Computing utility scores yet.
- Building `context stats` or broad analysis surfaces yet.
- Persisting speculative reasoning or transcript-like scratch state.
- Logging every expansion or downstream action in the first wave.

## Requirements

- Persist packet metadata for each compiled packet associated with the active session when one exists.
- Record enough packet metadata to support later analysis, including:
  - packet hash or deterministic identity
  - task summary
  - included item IDs
  - included anchors
  - inclusion reasons
  - compile timestamp
- Support compile recording even when packet generation is repeated within one session.
- Avoid storing full expanded source bodies as part of the packet record.
- Keep the stored representation inspectable and migration-friendly.
- Leave room for later linkage to verification and durable note updates.

## UX / Flows

Session-backed compile flow:
1. User starts or validates a session.
2. User runs `brain context compile`.
3. Brain emits the packet and records the packet metadata in session state.
4. Later session-close and telemetry work can refer back to the recorded packet.

No-session compile flow:
1. User runs `brain context compile --task "..."` without an active session.
2. Brain emits the packet normally.
3. Brain may skip persistent recording or write a minimal transient record, but it must not require a session to compile.

## Data / Interfaces

Suggested first-wave session packet fields:
- `packet_hash`
- `task_text`
- `task_source`
- `compiled_at`
- `included_item_ids`
- `included_anchors`
- `inclusion_reasons`

Recording principles:
- record facts, not chain-of-thought
- record IDs and anchors, not full source copies
- keep packet history append-only within the session where practical

## Risks / Open Questions

- Should no-session compile calls write a lightweight standalone record, or stay purely ephemeral until telemetry work arrives?
- How much packet history should survive in the active session file before storage becomes noisy?
- Is packet hash alone enough to compare repeated compiles, or should Brain also store a normalized packet signature?

## Rollout

1. Extend session state for packet recording.
2. Record packet metadata from `context compile`.
3. Add tests that prove packet recording is deterministic and inspectable.
4. Keep the output internal for now, ready for later analysis surfaces.

## Story Breakdown

- [ ] Extend Session State For Packet Records
- [ ] Record Packet Composition During Context Compile
- [ ] Add Deterministic Packet Recording Tests And No-Session Fallback Behavior

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]
- [[.brain/planning/specs/live-work-context.md]]

## Notes

Treat this epic as the floor for later telemetry, not the telemetry system itself.
