---
created: "2026-04-15T05:34:09Z"
epic: v3-utility-aware-context-ranking
project: brain
spec: v3-utility-aware-context-ranking
status: done
title: Surface Utility-Aware Reasons And Threshold Diagnostics
type: story
updated: "2026-04-15T15:26:09Z"
---
# Surface Utility-Aware Reasons And Threshold Diagnostics

Created: 2026-04-15T05:34:09Z

## Description

Expose utility-aware selection adjustments through explain diagnostics and explicit thresholds so ranking changes remain debuggable and do not feel magical.


## Acceptance Criteria

- [ ] Explain surfaces can say when an item was boosted or suppressed because of observed utility or noise signals
- [ ] Thresholds prevent single noisy sessions or very small samples from changing packet ranking too early
- [ ] Tests or diagnostics make it possible to inspect why a utility-aware ranking change did or did not activate




## Resources

- [[.brain/planning/specs/v3-utility-aware-context-ranking.md]]
- [[.brain/planning/specs/v3-context-utility-analysis-surfaces.md]]



## Notes
