---
created: "2026-04-15T03:55:57Z"
epic: v2-repo-boundary-graph
project: brain
status: approved
title: V2 Repo Boundary Graph Spec
type: spec
updated: "2026-04-15T12:00:00Z"
---
# V2 Repo Boundary Graph Spec

Created: 2026-04-15T03:40:00Z

## Why

Context packets get much better when Brain can reason from real repo boundaries. The existing structural repo work already identifies useful structure, but the compiler now needs a normalized boundary graph it can query directly.

## Problem

Current structural output is useful for humans and targeted inspection, but it is not yet shaped as a direct compiler input for file-to-boundary, package-to-test, and adjacency-aware context selection.

## Goals

- Build a normalized boundary model that the compiler can consume directly.
- Map files to boundaries and boundaries to likely tests.
- Reuse existing structural derivation work instead of replacing it.
- Keep the model inspectable and deterministic.

## Non-Goals

- Building a broad relationship-graph platform for every future Brain feature.
- Supporting arbitrary language ecosystems in the first pass.
- Adding telemetry or learning behavior here.
- Replacing the human-facing structural context surfaces.

## Requirements

- Normalize the current structural derivation into boundary records the compiler can query.
- Capture at least these first-wave relations:
  - file -> boundary
  - boundary -> likely tests
  - boundary -> adjacent boundaries
- Preserve boundary labels, roles, and primary responsibilities when derivable.
- Keep generation deterministic and refreshable from repo state.
- Make boundary data available to later context-selection work without requiring a full external index system.

## UX / Flows

Compiler consumption flow:
1. Brain detects changed files or task-targeted files.
2. Brain resolves those files into normalized boundaries.
3. Brain uses the resulting boundary set to bias packet selection and test hints.

Structural inspection flow:
1. User inspects the structural repo context.
2. Brain shows consistent boundary labels and relations that match what packet compilation will later use.

## Data / Interfaces

Suggested boundary fields:
- `id`
- `label`
- `role`
- `root_path`
- `files`
- `adjacent_boundaries`
- `owned_tests`
- `responsibilities`

First-wave relation sources:
- directory and package shape
- existing structural scanner output
- naming conventions for tests and package adjacency

## Risks / Open Questions

- How repo-specific can the first-wave model be before it becomes brittle?
- Should owned tests be strict or best-effort in the first pass?
- How much responsibility text belongs in the normalized model versus the human-facing structural output?

## Rollout

1. Normalize existing structural output into boundary records.
2. Add file-to-boundary and boundary-to-test relations.
3. Expose the normalized model to the compiler pipeline.
4. Keep existing structural inspection UX working throughout.

## Story Breakdown

- [x] Normalize Structural Output Into Compiler Boundary Records
- [x] Build File To Boundary And Boundary To Test Relations
- [x] Expose Boundary Adjacency And Responsibilities To Compiler Consumers

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/structural-repo-context.md]]
- [[.brain/planning/specs/live-work-context.md]]

## Notes

This epic should make structure machine-usable for packet assembly while keeping it easy for humans to inspect and debug.
