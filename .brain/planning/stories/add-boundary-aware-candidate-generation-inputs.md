---
created: "2026-04-15T05:33:26Z"
epic: v2-boundary-aware-context-selection
project: brain
spec: v2-boundary-aware-context-selection
status: todo
title: Add Boundary-Aware Candidate Generation Inputs
type: story
updated: "2026-04-15T05:33:26Z"
---
# Add Boundary-Aware Candidate Generation Inputs

Created: 2026-04-15T05:33:26Z

## Description

Teach the compiler to use touched boundaries, changed-file pressure, and boundary overlap as first-class candidate-generation and scoring inputs instead of relying mostly on lexical task matching.


## Acceptance Criteria

- [ ] Candidate generation uses normalized boundary data as a first-wave signal alongside task and live-work inputs
- [ ] Changed-file overlap and boundary overlap become explicit deterministic selection inputs
- [ ] Compiler output can later explain when an item was selected because it matched touched boundaries rather than generic task terms




## Resources

- [[.brain/planning/specs/v2-boundary-aware-context-selection.md]]


## Notes
