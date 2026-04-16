---
created: "2026-04-16T02:07:44Z"
epic: derived-doc-capsules-and-drift-audit
project: brain
spec: derived-doc-capsules-and-drift-audit
status: todo
title: Add Capsule Drift Audit And Doctor Diagnostics
type: story
updated: "2026-04-16T02:15:58Z"
---
# Add Capsule Drift Audit And Doctor Diagnostics

Created: 2026-04-16T02:07:44Z

## Description

Make stale or missing capsules visible through an audit surface so Brain can trust capsule-backed compile results without silently serving drifted summaries.

## Acceptance Criteria

- [ ] Brain reports `current`, `missing`, `stale`, and any first-wave coverage mismatch state for capsule-backed docs through `brain doctor`, a dedicated audit command, or both.
- [ ] When a capsule is stale or missing, compiler-facing diagnostics make that state visible and avoid silently treating the capsule as trustworthy.
- [ ] Audit coverage includes stricter checklist-style validation for any docs that opt into explicit coverage expectations.

## Resources

- [[.brain/planning/specs/derived-doc-capsules-and-drift-audit.md]]

## Notes
