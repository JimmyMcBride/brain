---
created: "2026-04-15T05:32:39Z"
epic: v1-context-packet-compiler
project: brain
spec: v1-context-packet-compiler
status: done
title: Add Context Compile Command And Stable Packet Schema
type: story
updated: "2026-04-15T14:10:41Z"
---
# Add Context Compile Command And Stable Packet Schema

Created: 2026-04-15T05:32:39Z

## Description

Add the first compiler-oriented context command and lock the stable packet contract so Brain can emit a compact working-set packet from explicit task text or the active session task.


## Acceptance Criteria

- [ ] `brain context compile` resolves task text from `--task` or the active session and fails clearly when neither exists
- [ ] Human and JSON output follow one stable packet schema with task framing, base contract, working set, verification hints, ambiguities, and provenance fields
- [ ] The compiler surface lands under the existing context command family without breaking current context commands




## Resources

- [[.brain/planning/specs/v1-context-packet-compiler.md]]

## Notes
