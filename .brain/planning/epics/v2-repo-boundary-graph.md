---
created: "2026-04-15T03:55:57Z"
project: brain
spec: v2-repo-boundary-graph
title: V2 Repo Boundary Graph
type: epic
updated: "2026-04-15T03:59:02Z"
---
# V2 Repo Boundary Graph

Created: 2026-04-15T03:40:00Z

## Summary

Extend Brain's existing structural repo understanding into a normalized boundary graph that can drive packet selection with file, package, module, and test relationships instead of mostly lexical heuristics.

## Why It Matters

The compiler becomes materially sharper once it knows the repo's real boundaries. This epic turns structure into a reusable graph-like substrate for context selection rather than leaving it as mostly human-readable output.

## Spec

- [[.brain/planning/specs/v2-repo-boundary-graph.md]]

## Sources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/structural-repo-context.md]]
- [[.brain/planning/specs/live-work-context.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

Reuse the structural repo context work that already exists. `v2` should normalize and deepen it for compiler use rather than starting a separate graph platform from scratch.
