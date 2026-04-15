---
created: "2026-04-15T03:55:57Z"
epic: v3-context-packet-telemetry
project: brain
status: approved
title: V3 Context Packet Telemetry Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V3 Context Packet Telemetry Spec

Created: 2026-04-15T03:40:00Z

## Why

Once Brain emits deterministic packets, the next step is to learn from real usage. That requires a local telemetry layer that records what was included, what was expanded, and how the session turned out.

## Problem

`v1` packet recording is enough to preserve basic provenance, but it is not enough to answer whether included items were useful, whether users expanded them, or whether certain packet shapes correlate with successful verification and durable updates.

## Goals

- Record packet usage and outcome signals locally.
- Capture enough detail to support future utility analysis and ranking.
- Keep telemetry inspectable and privacy-preserving.
- Avoid polluting durable memory with ephemeral scratch or chain-of-thought.

## Non-Goals

- Aggressive automatic ranking changes in this epic.
- Hosted analytics or remote telemetry.
- Storing full reasoning traces.
- Turning all session activity into durable memory.

## Requirements

- Record included packet item IDs for compiled packets.
- Record expanded item IDs when users or later commands expand from summaries to full anchors.
- Link packet usage to:
  - verification commands
  - verification success or failure
  - durable note updates when they occur
  - session closeout status
- Keep telemetry local-first and inspectable.
- Make telemetry storage migration-friendly and bounded.
- Distinguish between packet composition, packet expansion, and packet outcome events.

## UX / Flows

Telemetry capture flow:
1. User compiles a packet.
2. Brain records the included items and packet identity.
3. User expands some items or runs verification commands.
4. Brain links those later events back to the compiled packet where possible.

Session closeout linkage flow:
1. A session reaches closeout.
2. Brain knows which packets were compiled and which items were later relevant to verification or durable updates.
3. That data becomes available for later utility analysis.

## Data / Interfaces

Suggested telemetry event families:
- `packet_compiled`
- `item_expanded`
- `verification_recorded`
- `durable_update_recorded`
- `session_closed`

Suggested stored fields:
- `session_id`
- `packet_hash`
- `task_summary`
- `included_item_ids`
- `expanded_item_ids`
- `verification_commands`
- `verification_success`
- `durable_updates_written`
- `closeout_status`

## Risks / Open Questions

- How much event detail is enough before local state becomes noisy?
- Should expansion events be recorded only for compiler-aware commands, or also inferred from later file reads?
- How should telemetry retention or compaction work for long-lived repos?

## Rollout

1. Add local telemetry storage or session-backed event persistence.
2. Record packet compilation, expansion, and outcome events.
3. Add tests that prove linkage between packet usage and later session results.
4. Prepare the data model for later analysis and ranking epics.

## Story Breakdown

- [ ] Add Local Packet Telemetry Event Model And Storage
- [ ] Record Packet Expansion Verification And Closeout Linkage
- [ ] Add Bounded Telemetry Retention And Linkage Tests

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v1-packet-provenance-and-session-recording.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Notes

If this epic starts to look like generic analytics, it has drifted out of scope. The only reason it exists is to improve compiler usefulness.
