---
created: "2026-04-15T05:32:39Z"
epic: v1-base-contract-and-summary-anchors
project: brain
spec: v1-base-contract-and-summary-anchors
status: todo
title: Generate Summary And Anchor Forms For Major Context Sources
type: story
updated: "2026-04-15T05:32:39Z"
---
# Generate Summary And Anchor Forms For Major Context Sources

Created: 2026-04-15T05:32:39Z

## Description

Turn the major Brain-owned context sources into lossy summaries with exact anchors so later compiler packets can include compact context by default and expand only when needed.


## Acceptance Criteria

- [ ] Major startup and durable context sources produce summary-plus-anchor forms instead of requiring full-document inclusion by default
- [ ] Generated summaries retain exact anchor paths and sections so later expansion is deterministic and inspectable
- [ ] The summary-generation path is tested against the major Brain-owned context sources used by startup and early packet assembly




## Resources

- [[.brain/planning/specs/v1-base-contract-and-summary-anchors.md]]

## Notes
