---
container: core-product-tightening-and-simplification
created: "2026-04-11T14:49:34Z"
epic: core-product-tightening-and-simplification
project: brain
spec: core-product-tightening-and-simplification
status: done
title: Make Policy Overrides Behave Like Real Overrides
type: story
updated: "2026-04-11T21:53:09Z"
---
# Make Policy Overrides Behave Like Real Overrides

Created: 2026-04-11T14:49:34Z

## Description

Fix .brain/policy.override.yaml semantics so booleans can be turned both on and off, not only on. Keep merge behavior explicit and testable for scalar fields, lists, and booleans.


## Acceptance Criteria

- [ ] An override can disable require_task, single_active, require_brain_doctor, and require_memory_update_on_repo_change
- [ ] Policy tests cover both enable and disable cases



## Resources

- internal/projectcontext/policy.go
- internal/session/manager.go
- [[.brain/planning/specs/core-product-tightening-and-simplification.md]]




## Notes
