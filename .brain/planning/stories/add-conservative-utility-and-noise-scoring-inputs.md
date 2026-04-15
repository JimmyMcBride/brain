---
created: "2026-04-15T05:34:09Z"
epic: v3-utility-aware-context-ranking
project: brain
spec: v3-utility-aware-context-ranking
status: todo
title: Add Conservative Utility And Noise Scoring Inputs
type: story
updated: "2026-04-15T05:34:09Z"
---
# Add Conservative Utility And Noise Scoring Inputs

Created: 2026-04-15T05:34:09Z

## Description

Introduce the first utility-aware ranking inputs so Brain can reward repeated signal and suppress repeated noise without replacing the deterministic compiler model.


## Acceptance Criteria

- [ ] Ranking can incorporate utility-oriented inputs such as repeated expansion, successful verification linkage, and repeated low-impact inclusion
- [ ] Utility and noise signals activate conservatively rather than dominating ranking immediately
- [ ] The scoring inputs stay local and repo-specific rather than implying cross-repo learning




## Resources

- [[.brain/planning/specs/v3-utility-aware-context-ranking.md]]


## Notes
