---
created: "2026-04-14T00:14:43Z"
epic: structural-repo-context
project: brain
spec: structural-repo-context
status: done
title: Add Structural Context Storage And Status Surface
type: story
updated: "2026-04-14T00:14:49Z"
---
# Add Structural Context Storage And Status Surface

Created: 2026-04-14T00:14:43Z

## Description

Add the derived structural storage model in the project SQLite database plus the status command that reports freshness and counts without mutating cache state.


## Acceptance Criteria

- [ ] Structural repo context stores state and items in .brain/state/brain.sqlite3 using the approved schema
- [ ] brain context structure status reports missing, stale, and fresh states with structural freshness metadata
- [ ] Structural freshness remains independent from markdown-search freshness




## Resources

- [[.brain/planning/specs/structural-repo-context.md]]



## Notes
