---
created: "2026-04-14T00:28:50Z"
epic: live-work-context
project: brain
spec: live-work-context
status: done
title: Implement Changed File Boundary And Nearby Test Signals
type: story
updated: "2026-04-14T00:37:30Z"
---
# Implement Changed File Boundary And Nearby Test Signals

Created: 2026-04-14T00:28:50Z

## Description

Derive changed files, touched structural boundaries, and nearby tests on demand from git state, session baseline, and the structural repo context layer.


## Acceptance Criteria

- [ ] Changed-file detection unions session-baseline commit changes and current worktree status when a session exists
- [ ] Touched boundaries resolve from structural repo context for changed paths
- [ ] Nearby tests cover direct test changes, same-directory test files, and touched-boundary test surfaces




## Resources

- [[.brain/planning/specs/live-work-context.md]]


## Notes
