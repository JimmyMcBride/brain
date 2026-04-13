---
created: "2026-04-13T22:00:47Z"
project: brain
spec: task-context-assembly
title: Task Context Assembly
type: epic
updated: "2026-04-13T22:02:39Z"
---
# Task Context Assembly

Created: 2026-04-13T22:00:47Z

## Summary

Make Brain better at assembling the right context for an active task from typed local sources instead of relying on static context bundles plus note retrieval alone.

## Why It Matters

This is the clearest product gap after the recent retrieval and context-loading work. Brain already manages truth well, but it still needs a stronger answer to: what should the agent read right now, and why those sources?

## Spec

- [[.brain/planning/specs/task-context-assembly.md]]

## Sources

- [[.brain/brainstorms/context-engine-v2.md]]
- [[.brain/planning/specs/retrieval-and-index-lifecycle.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Progress

- Draft spec created.
- Story breakdown intentionally deferred until the spec direction is approved.

## Notes

Prefer extending the existing context and search surfaces first. A dedicated packet command is a follow-on only if the assembled output proves distinct enough to deserve its own primary UX.
