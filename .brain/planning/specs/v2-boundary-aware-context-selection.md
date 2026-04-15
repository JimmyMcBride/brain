---
created: "2026-04-15T03:55:57Z"
epic: v2-boundary-aware-context-selection
project: brain
status: approved
title: V2 Boundary-Aware Context Selection Spec
type: spec
updated: "2026-04-15T12:00:00Z"
---
# V2 Boundary-Aware Context Selection Spec

Created: 2026-04-15T03:40:00Z

## Why

The next meaningful gain after `v1` is sharper selection. Brain should prefer context tied to the actual boundaries under change instead of leaning mostly on lexical matching and generic task wording.

## Problem

`v1` packet assembly can produce compact justified output, but it still risks selecting broad or weakly relevant context because it lacks strong structural overlap signals and note-to-boundary linkage.

## Goals

- Improve packet relevance using boundary overlap and changed-file pressure.
- Link durable notes and generated context to normalized boundaries where possible.
- Keep selection reasons inspectable and deterministic.
- Improve multi-boundary task handling without large token growth.

## Non-Goals

- Utility-based reranking from telemetry.
- Broad semantic ranking systems as the primary selector.
- Hiding ranking behavior behind opaque scores.
- Turning every note into a densely annotated graph node immediately.

## Requirements

- Use normalized boundary data as a first-class candidate generation and scoring signal.
- Weight changed-file overlap and boundary overlap explicitly in candidate selection.
- Add note-to-boundary linkage for durable notes and generated context items where possible.
- Surface boundary-aware inclusion reasons in compiler output.
- Handle tasks that touch multiple boundaries without flooding the packet.
- Preserve deterministic packet assembly for the same repo state and task input.

## UX / Flows

Changed-code flow:
1. User works in a repo with changed files.
2. Brain compiles a packet.
3. Brain identifies touched boundaries and selects context tied to those boundaries.
4. Brain explains that items were included because of boundary overlap or changed-file pressure.

Multi-boundary task flow:
1. User asks for a change spanning multiple packages or modules.
2. Brain identifies multiple high-pressure boundaries.
3. Brain includes the minimum justified context from each instead of overloading one boundary or flooding all neighbors.

## Data / Interfaces

Additional first-wave selection signals in this epic:
- `boundary_overlap`
- `changed_file_overlap`
- `boundary_recency` when derivable from current worktree or session baseline
- `note_boundary_match`

Per-item diagnostics should support reasons such as:
- `contains changed files`
- `adjacent to touched boundary`
- `linked durable note for touched boundary`

## Risks / Open Questions

- How much note annotation is needed to make note-to-boundary linkage useful without creating maintenance burden?
- Should adjacency be a weak bonus only, or can it be a primary inclusion reason when direct overlap is absent?
- How should Brain suppress noisy neighbors in large directories or packages?

## Rollout

1. Add boundary linkage to candidate generation.
2. Blend changed-file and boundary-overlap signals into deterministic selection.
3. Add explicit boundary-aware inclusion reasons.
4. Tune packet caps for multi-boundary tasks without changing the overall packet mental model.

## Story Breakdown

- [x] Add Boundary-Aware Candidate Generation Inputs
- [x] Link Durable Notes And Generated Context To Boundaries
- [x] Balance Multi-Boundary Packets And Surface Boundary Diagnostics

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v2-repo-boundary-graph.md]]
- [[.brain/planning/specs/task-context-assembly.md]]
- [[.brain/planning/specs/retrieval-and-index-lifecycle.md]]

## Notes

This epic should make Brain feel sharper without making it mysterious.
