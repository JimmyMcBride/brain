---
created: "2026-04-15T05:32:39Z"
epic: v1-context-packet-compiler
project: brain
spec: v1-context-packet-compiler
status: done
title: Teach Docs And The Brain Skill About Context Compile
type: story
updated: "2026-04-15T14:10:41Z"
---
# Teach Docs And The Brain Skill About Context Compile

Created: 2026-04-15T05:32:39Z

## Description

Add additive guidance for the new compiler surface in user docs and the Brain skill so agents can discover `context compile` early without losing compatibility guidance for the existing context surfaces.


## Acceptance Criteria

- [ ] `docs/usage.md` introduces `brain context compile` as a new compiler-oriented surface without falsely claiming older context paths are gone
- [ ] `skills/brain/SKILL.md` teaches when to use `context compile` and how it relates to existing context commands
- [ ] The new guidance keeps the migration additive in `v1` rather than teaching `context compile` as the only valid path yet




## Resources

- [[.brain/planning/specs/v1-context-packet-compiler.md]]
- [[skills/brain/SKILL.md]]
- [[docs/usage.md]]

## Notes
