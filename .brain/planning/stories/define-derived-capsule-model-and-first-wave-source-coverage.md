---
created: "2026-04-16T02:07:44Z"
epic: derived-doc-capsules-and-drift-audit
project: brain
spec: derived-doc-capsules-and-drift-audit
status: todo
title: Define Derived Capsule Model And First-Wave Source Coverage
type: story
updated: "2026-04-16T05:39:00Z"
---
# Define Derived Capsule Model And First-Wave Source Coverage

Created: 2026-04-16T02:07:44Z

## Description

Define the derived capsule record and lock the first-wave source-doc coverage so Brain has a bounded, inspectable capsule system instead of an open-ended rule-engine surface.

## Acceptance Criteria

- [ ] Capsule records define source path, source anchor or section, source hash, capsule content, estimated token cost, and any first-wave coverage metadata Brain needs.
- [ ] The capsule model reuses or cleanly specializes the existing summary-and-anchor pipeline rather than introducing a second parallel summary abstraction.
- [ ] The first-wave source set is explicitly limited to the highest-value Brain-managed docs still causing packet pressure after budgets and reuse.
- [ ] The design keeps full markdown docs canonical and treats capsules as derived compiler inputs, not a second truth system.
- [ ] The design defines explicit capsule usage modes of `off`, `auto`, and `on`, with default `off`, so later implementation starts from a constrained operator-facing surface instead of an ambient hidden feature.

## Resources

- [[.brain/planning/specs/derived-doc-capsules-and-drift-audit.md]]

## Notes
