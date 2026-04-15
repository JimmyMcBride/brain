---
created: "2026-04-15T05:33:26Z"
epic: v2-verification-and-test-surface-derivation
project: brain
spec: v2-verification-and-test-surface-derivation
status: done
title: Derive Nearby Tests From Changed Files And Boundaries
type: story
updated: "2026-04-15T12:00:00Z"
---
# Derive Nearby Tests From Changed Files And Boundaries

Created: 2026-04-15T05:33:26Z

## Description

Promote nearby tests into a structured compiler surface by deriving them from changed files and normalized boundaries instead of only surfacing incidental test hints.


## Acceptance Criteria

- [ ] The compiler can derive nearby-test candidates from changed files and normalized boundaries
- [ ] Nearby-test items include enough relation and reason metadata to explain why they were selected
- [ ] The first-wave nearby-test derivation remains bounded and avoids dumping broad test surfaces into the default packet




## Resources

- [[.brain/planning/specs/v2-verification-and-test-surface-derivation.md]]


## Notes
