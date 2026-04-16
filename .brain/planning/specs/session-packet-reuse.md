---
created: "2026-04-16T04:37:00Z"
epic: session-packet-reuse
project: brain
status: done
title: Session Packet Reuse Spec
type: spec
updated: "2026-04-16T04:39:24Z"
---
# Session Packet Reuse Spec

Created: 2026-04-16T04:37:00Z

## Why

Brain already records compiled packets and later outcomes inside sessions, but every compile is still treated like a fresh packet response. Long coding sessions often revisit the same task with the same boundaries and only small worktree changes. Brain should reuse what is still valid instead of paying the full packet cost every turn.

## Problem

The current compiler has packet history, but not packet reuse. That means Brain cannot tell the user or agent that the previous packet is still valid, cannot emit deltas relative to a prior packet, and cannot reduce repeated prompt weight during long-running work. Reused full-packet reprints would not meaningfully solve the token problem.

## Goals

- Add session-local packet reuse when the relevant compile inputs have not changed.
- Add delta metadata when the task is the same but the packet changed meaningfully.
- Keep reuse invalidation explicit and deterministic.
- Surface reuse lineage in packet output and packet explanation flows.
- Make reuse materially reduce repeated packet weight, not only compiler work.

## Non-Goals

- Cross-session or cross-project packet reuse.
- Hidden reuse that makes packet provenance harder to understand.
- Semantic diffing of free-form reasoning.
- Partial packet streaming or remote prompt transport concerns.

## Requirements

- Brain must compute a reuse fingerprint from the compile inputs that actually matter, such as:
  - task text or task summary
  - task source
  - relevant changed files
  - touched boundaries
  - source summary hashes or equivalent doc-state inputs
  - policy-driven verification requirements
- If the fingerprint matches the latest active-session packet, `brain context compile` must be able to return that packet as reused instead of rebuilding it.
- If the task is stable but the fingerprint changed, Brain must expose delta metadata linking the new packet to the previous one.
- Users must be able to force a fresh compile when debugging with a flag such as `--fresh`.
- `brain context explain --last` must make reuse or delta lineage visible.
- Packet reuse must not weaken deterministic behavior or hide when the underlying repo state changed.
- The default session-local reuse and delta paths must avoid re-emitting unchanged packet sections wholesale when Brain can safely refer to the previously compiled packet.
- If Brain falls back to a full packet because no reusable base packet is available or standalone output is required, that fallback should be explicit.

## UX / Flows

Reuse flow:
1. User runs `brain context compile --task "tighten auth flow"`.
2. Brain finds a matching fingerprint in the active session.
3. Brain returns a compact reused response tied to the prior packet instead of rebuilding and reprinting the whole packet.

Delta flow:
1. User runs `brain context compile` again after editing files in the same task.
2. Brain detects the same task but changed compile inputs.
3. Brain returns a compact delta tied to the previous packet, such as changed sections, changed item ids, or invalidation reasons.

Forced refresh:
1. User runs `brain context compile --fresh`.
2. Brain bypasses reuse and emits a new full packet even if the fingerprint matches.

## Data / Interfaces

Add packet metadata:
- `cache_status` (`fresh`, `reused`, `delta`)
- `reused_from`
- `delta_from`
- `changed_sections`
- `changed_item_ids`
- `fingerprint`
- `full_packet_included`

Potential CLI surface:
- `brain context compile --fresh`

Potential explain additions:
- reuse lineage
- delta lineage
- invalidation reason summary
- fallback reason when Brain had to emit a full packet

## Risks / Open Questions

- Which compile inputs should invalidate reuse immediately, and which only affect delta reporting?
- How much delta detail is enough before the output becomes as verbose as a full packet?
- When should Brain fall back to a full packet for standalone usability instead of a compact reuse or delta response?

## Rollout

1. Define compile fingerprints and reuse metadata.
2. Add active-session reuse detection plus compact reused responses to `context compile`.
3. Add compact delta metadata for changed-but-related packets.
4. Teach explain surfaces, docs, and the Brain skill about reuse, compact deltas, and forced refresh.

## Story Breakdown

- [x] Add Compile Fingerprint And Reuse Lineage To Session Packets
- [x] Reuse Latest Matching Session Packet And Support Fresh Bypass
- [x] Emit Delta Lineage And Invalidation Diagnostics For Related Packets
- [x] Teach Explain Docs And Brain Skill About Packet Reuse

## Resources

- [[.brain/brainstorms/token-efficient-context-direction.md]]
- [[.brain/planning/specs/v1-packet-provenance-and-session-recording.md]]
- [[.brain/planning/specs/v3-context-packet-telemetry.md]]
- [[.brain/planning/specs/v3-context-utility-analysis-surfaces.md]]

## Notes

This is the main defense against long-conversation token creep. Reuse should be obvious, inspectable, easy to bypass when debugging, and actually smaller on the wire than a fresh packet.

Implemented on `feature/context-packet-optimization` with deterministic compile fingerprints, session-recorded full packet bodies plus lineage metadata, compact `reused` and `delta` compile responses, the `--fresh` escape hatch, and explain or CLI surfaces that make invalidation plus fallback reasons explicit.
