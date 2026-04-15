---
created: "2026-04-15T05:33:26Z"
epic: v2-boundary-aware-context-selection
project: brain
spec: v2-boundary-aware-context-selection
status: todo
title: Link Durable Notes And Generated Context To Boundaries
type: story
updated: "2026-04-15T05:33:26Z"
---
# Link Durable Notes And Generated Context To Boundaries

Created: 2026-04-15T05:33:26Z

## Description

Add enough boundary linkage to durable notes and generated context items that the compiler can select them because they belong to a touched area of the repo, not only because they match the task lexically.


## Acceptance Criteria

- [ ] Durable notes and generated context items can carry boundary linkage where the repo provides enough structure to derive it
- [ ] Compiler selection can prefer linked context items for touched boundaries without requiring every note to be manually annotated
- [ ] Boundary linkage remains inspectable and maintainable instead of becoming a hidden or brittle metadata layer




## Resources

- [[.brain/planning/specs/v2-boundary-aware-context-selection.md]]


## Notes
