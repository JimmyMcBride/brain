---
created: "2026-04-13T06:23:52Z"
epic: planning-and-brainstorming-ux
project: brain
spec: planning-and-brainstorming-ux
status: in_progress
title: Add Session-Aware Memory Distillation
type: story
updated: "2026-04-13T07:24:23Z"
---
# Add Session-Aware Memory Distillation

Created: 2026-04-13T06:23:52Z

## Description

Add a first-class `brain distill` workflow that inspects recent session history, command runs, and git diff, then proposes durable note updates without silently writing them.

## Acceptance Criteria

- [ ] Manual distill can scope to the active session and gather recent command, diff, and note context
- [ ] Distill output proposes updates for Brain-managed note targets instead of auto-applying them
- [ ] Users or agents can review and edit candidate updates before accepting them

## Resources

- [[.brain/planning/specs/planning-and-brainstorming-ux.md]]
- [[.brain/brainstorms/mempalace-inspired-brain-improvements.md]]

## Notes
