---
created: "2026-04-15T03:55:57Z"
project: brain
spec: v2-boundary-aware-context-selection
title: V2 Boundary-Aware Context Selection
type: epic
updated: "2026-04-15T03:59:02Z"
---
# V2 Boundary-Aware Context Selection

Created: 2026-04-15T03:40:00Z

## Summary

Use repo boundaries, changed-file pressure, and note-to-boundary linkage to make packet selection substantially sharper than `v1`'s mostly deterministic first-wave heuristics.

## Why It Matters

`v1` proves the packet shape. `v2` proves that Brain can choose better context because it understands where work is happening in the codebase, not just what words appeared in the task.

## Spec

- [[.brain/planning/specs/v2-boundary-aware-context-selection.md]]

## Sources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v2-repo-boundary-graph.md]]
- [[.brain/planning/specs/task-context-assembly.md]]
- [[.brain/planning/specs/retrieval-and-index-lifecycle.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

This epic should improve selection quality without introducing opaque ranking behavior. Boundary overlap and changed-file pressure should remain inspectable selection reasons.
