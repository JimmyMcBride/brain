---
created: "2026-04-13T23:17:25Z"
epic: task-context-assembly
project: brain
spec: task-context-assembly
status: todo
title: Add Explain Mode, Omitted Nearby Context, And Confidence Reporting
type: story
updated: "2026-04-13T23:17:25Z"
---
# Add Explain Mode, Omitted Nearby Context, And Confidence Reporting

Created: 2026-04-13T23:17:25Z

## Description

Expand task-context assembly with explain-mode rationale, omitted-nearby candidates, missing-group reporting, ambiguities, and confidence buckets.


## Acceptance Criteria

- [ ] `brain context assemble --explain` includes rationale and omitted-nearby sections without changing default compact output
- [ ] Ambiguities and confidence are computed from the approved assembly rules
- [ ] Existing `brain context load` and `brain search --inject` behaviors remain unchanged




## Resources

- [[.brain/planning/specs/task-context-assembly.md]]



## Notes
