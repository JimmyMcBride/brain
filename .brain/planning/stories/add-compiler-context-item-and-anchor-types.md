---
created: "2026-04-15T05:32:39Z"
epic: v1-base-contract-and-summary-anchors
project: brain
spec: v1-base-contract-and-summary-anchors
status: todo
title: Add Compiler Context Item And Anchor Types
type: story
updated: "2026-04-15T05:32:39Z"
---
# Add Compiler Context Item And Anchor Types

Created: 2026-04-15T05:32:39Z

## Description

Define the shared compiler-facing context item, anchor, and related packet-selection primitives so later packet assembly can operate on stable summary-first inputs instead of ad hoc document payloads.


## Acceptance Criteria

- [ ] Compiler-facing types cover reusable item identity, item kind, summary content, anchor path and section, and lightweight boundary or file linkage
- [ ] The first-wave type model can represent base-contract, durable-note, generated-context, workflow-rule, and verification-recipe items without special-case hacks
- [ ] The new types are placed where projectcontext, taskcontext, livecontext, and session code can adopt them incrementally without circular package coupling




## Resources

- [[.brain/planning/specs/v1-base-contract-and-summary-anchors.md]]

## Notes
