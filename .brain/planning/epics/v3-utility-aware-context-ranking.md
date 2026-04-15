---
created: "2026-04-15T03:55:57Z"
project: brain
spec: v3-utility-aware-context-ranking
title: V3 Utility-Aware Context Ranking
type: epic
updated: "2026-04-15T04:01:06Z"
---
# V3 Utility-Aware Context Ranking

Created: 2026-04-15T03:40:00Z

## Summary

Blend packet utility signals into candidate selection so Brain can prefer context that repeatedly proves useful while still keeping ranking deterministic and debuggable.

## Why It Matters

This is the point where Brain starts learning from actual usefulness instead of relying only on static heuristics. Done carefully, it should improve packet quality without making the compiler feel opaque.

## Spec

- [[.brain/planning/specs/v3-utility-aware-context-ranking.md]]

## Sources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v3-context-packet-telemetry.md]]
- [[.brain/planning/specs/v3-context-utility-analysis-surfaces.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

Keep the first ranking model simple. The goal is not to build a vibes engine; it is to bias deterministic selection with observed utility while preserving inspectability.
