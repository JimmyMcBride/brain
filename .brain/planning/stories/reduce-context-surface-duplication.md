---
container: core-product-tightening-and-simplification
created: "2026-04-11T14:49:34Z"
epic: core-product-tightening-and-simplification
project: brain
spec: core-product-tightening-and-simplification
status: todo
title: Reduce Context Surface Duplication
type: story
updated: "2026-04-11T21:53:09Z"
---
# Reduce Context Surface Duplication

Created: 2026-04-11T14:49:34Z

## Description

Revisit the current model that generates overlapping content into root docs, .brain/context, and agent wrappers. Reduce duplication where it is not buying clarity while keeping deterministic refresh behavior.


## Acceptance Criteria

- [ ] Each generated file has a distinct purpose
- [ ] Near-duplicate context and doc surfaces are reduced or clearly justified
- [ ] Context refresh remains deterministic




## Resources

- internal/projectcontext/manager.go
- docs/project-overview.md
- .brain/context/overview.md
- [[.brain/planning/specs/core-product-tightening-and-simplification.md]]





## Notes
