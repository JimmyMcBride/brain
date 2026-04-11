---
container: core-product-tightening-and-simplification
created: "2026-04-11T14:49:34Z"
project: brain
status: todo
title: Keep Find Simple Or Make It Scale Deliberately
type: story
updated: "2026-04-11T14:49:34Z"
---
# Keep Find Simple Or Make It Scale Deliberately

Created: 2026-04-11T14:49:34Z

## Description

Decide whether find should remain a simple filesystem scan for small repos or gain indexed acceleration for larger repos. Do not half-optimize it; either document it as the lightweight exact-ish finder or explicitly move it onto indexed metadata search.


## Acceptance Criteria

- [ ] The intended role of find is clear in code and docs
- [ ] Large-repo performance expectations are not ambiguous



## Resources

- cmd/find.go
- internal/notes/manager.go



## Notes
