---
created: "2026-04-15T03:55:57Z"
project: brain
spec: v2-verification-and-test-surface-derivation
title: V2 Verification And Test Surface Derivation
type: epic
updated: "2026-04-15T03:59:02Z"
---
# V2 Verification And Test Surface Derivation

Created: 2026-04-15T03:40:00Z

## Summary

Derive nearby tests and verification recipes as structured compiler inputs so packets can tell the agent how to verify likely changes, not just what code and notes to read.

## Why It Matters

A useful packet does not stop at facts. It should also say how the result should be verified. This epic turns test ownership and verification recipes into first-class packet surfaces instead of incidental output.

## Spec

- [[.brain/planning/specs/v2-verification-and-test-surface-derivation.md]]

## Sources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/live-work-context.md]]
- [[.brain/planning/specs/structural-repo-context.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

Keep verification hints grounded in observable repo and session signals. This should improve packet usefulness without adding opaque policy magic.
