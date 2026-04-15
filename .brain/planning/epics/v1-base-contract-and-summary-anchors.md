---
created: "2026-04-15T03:55:57Z"
project: brain
spec: v1-base-contract-and-summary-anchors
title: V1 Base Contract And Summary Anchors
type: epic
updated: "2026-04-15T03:59:02Z"
---
# V1 Base Contract And Summary Anchors

Created: 2026-04-15T03:40:00Z

## Summary

Define the compact always-on base contract and the summary-plus-anchor representation that turns Brain's major context sources into small, selectable compiler inputs instead of large preloaded documents.

## Why It Matters

The context compiler cannot emit small useful packets until Brain has trustworthy, compressed building blocks. This epic creates those building blocks and shrinks the boot path so Brain can start from tiny summaries and expand only when needed.

## Spec

- [[.brain/planning/specs/v1-base-contract-and-summary-anchors.md]]

## Sources

- [[.brain/brainstorms/context-substrate-direction.md]]
- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/task-context-assembly.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

Keep the base contract aggressively small. This epic succeeds only if Brain's default include gets smaller, clearer, and more packet-friendly than today's document-centric startup path.
