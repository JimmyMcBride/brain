---
created: "2026-04-15T03:55:57Z"
epic: v1-context-packet-compiler
project: brain
status: approved
title: V1 Context Packet Compiler Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V1 Context Packet Compiler Spec

Created: 2026-04-15T03:40:00Z

## Why

Brain needs a primary context surface that compiles the smallest justified working set for the current task. The first wave should prove that deterministic packet assembly materially beats the current document-centric startup flow without requiring a large rewrite.

## Problem

Current Brain context surfaces still require the agent to do too much assembly work after context is loaded. Even with live-work and task-assembly improvements, Brain does not yet emit one compact packet that says what matters next and why.

## Goals

- Add `brain context compile` as the compiler-oriented task packet surface.
- Emit a deterministic summary-first packet for the current task.
- Select from base contract items, live-work signals, and durable note summaries.
- Attach inclusion reasons to every selected item.
- Keep the packet small enough to be practical for agent startup.
- Preserve compatibility with the existing context surfaces during the migration.

## Non-Goals

- Adaptive reranking from telemetry.
- A full standalone context-index subsystem in the first wave.
- Replacing all older `context` commands immediately.
- Expanding every selected item into full source content by default.

## Requirements

- Add `brain context compile --task "..."` and `brain context compile` using the active session task.
- Support `--json` output with a stable packet contract.
- Include packet sections for:
  - task summary
  - base contract
  - working set boundaries/files/tests/notes
  - risks or ambiguities when present
  - verification hints
  - provenance or inclusion reasons
- Build the first-wave packet from:
  - base-contract items
  - changed files
  - touched boundaries
  - nearby tests
  - top durable note summaries
- Default to summaries rather than full source bodies.
- Keep packet assembly deterministic for the same repo state and task input.
- Fail clearly when no task is available from flags or session state.
- Update additive user guidance for the new compiler surface in `docs/usage.md` and `skills/brain/SKILL.md` without making `context compile` the only recommended context path yet.

## UX / Flows

Explicit task flow:
1. User runs `brain context compile --task "tighten auth flow" --json`.
2. Brain resolves the task and gathers first-wave candidates.
3. Brain emits a compact packet with summaries, anchors, and inclusion reasons.

Active session flow:
1. User starts or validates a session.
2. User runs `brain context compile`.
3. Brain uses the active session task as the packet target.
4. Brain emits the same packet shape and marks the task source accordingly.

## Data / Interfaces

First-wave packet sections:
- `task`
- `base_contract`
- `working_set.boundaries`
- `working_set.files`
- `working_set.tests`
- `working_set.notes`
- `verification`
- `ambiguities`
- `provenance`

Required per-item diagnostics:
- `id`
- `summary`
- `anchor`
- `reason`

First-wave candidate channels:
- base contract
- changed-file overlap
- touched boundaries
- nearby tests
- durable note summaries
- workflow-required items when strongly matched

## Risks / Open Questions

- How small should the default packet be before usefulness drops?
- Should the first-wave packet have an explicit `risks` section or rely on `ambiguities` plus provenance?
- Is a separate `context compile` command clearly better than evolving `context assemble`, or should they share implementation behind the scenes?

## Rollout

1. Define the stable packet schema.
2. Add the `context compile` command.
3. Wire the first-wave candidate gathering and deterministic selection path.
4. Add human and JSON output with inclusion reasons.
5. Keep existing context surfaces intact while `context compile` proves itself.

## Story Breakdown

- [ ] Add Context Compile Command And Stable Packet Schema
- [ ] Implement First-Wave Candidate Gathering And Deterministic Selection
- [ ] Render Packet Output With Provenance Ambiguities And Verification Hints
- [ ] Teach Docs And The Brain Skill About Context Compile

## Resources

- [[.brain/brainstorms/context-substrate-direction.md]]
- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/task-context-assembly.md]]
- [[.brain/planning/specs/live-work-context.md]]
- [[.brain/planning/specs/retrieval-and-index-lifecycle.md]]

## Notes

The compiler should be the new center of gravity, but not yet the only surface. Compatibility matters in `v1` because the architecture shift is large enough already.
