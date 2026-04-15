---
created: "2026-04-15T05:33:26Z"
epic: v2-repo-boundary-graph
project: brain
spec: v2-repo-boundary-graph
status: done
title: Build File To Boundary And Boundary To Test Relations
type: story
updated: "2026-04-15T12:00:00Z"
---
# Build File To Boundary And Boundary To Test Relations

Created: 2026-04-15T05:33:26Z

## Description

Add the first-wave structural relations that let the compiler map changed files into boundaries and those boundaries into likely nearby tests.


## Acceptance Criteria

- [ ] The boundary model supports deterministic file-to-boundary resolution for compiler use
- [ ] The first-wave graph includes boundary-to-test relations derived from observable repo structure and naming conventions
- [ ] The relation model is usable by compiler selection without requiring a separate external graph database or service




## Resources

- [[.brain/planning/specs/v2-repo-boundary-graph.md]]

## Notes
