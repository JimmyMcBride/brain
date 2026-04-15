---
created: "2026-04-15T03:55:58Z"
project: brain
spec: v4-promotion-gate-and-durable-promotion-types
title: V4 Promotion Gate And Durable Promotion Types
type: epic
updated: "2026-04-15T04:01:06Z"
---
# V4 Promotion Gate And Durable Promotion Types

Created: 2026-04-15T03:40:00Z

## Summary

Add a strict promotion gate that decides what ephemeral session findings are allowed to become durable memory and what should disappear with the session.

## Why It Matters

A stronger compiler will surface more ephemeral state. Without a disciplined promotion model, Brain risks polluting durable memory with transient reasoning and turning its memory layer into noise.

## Spec

- [[.brain/planning/specs/v4-promotion-gate-and-durable-promotion-types.md]]

## Sources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/brainstorms/context-substrate-direction.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Progress

- Approved spec in place.
- Story set created and ready for execution planning.

## Notes

This epic should make durable memory stricter and more useful, not easier to spam. Promotion is valuable only if it improves memory quality.
