---
created: "2026-04-16T04:38:00Z"
project: brain
spec: derived-doc-capsules-and-drift-audit
title: Derived Doc Capsules And Drift Audit
type: epic
updated: "2026-04-16T05:39:00Z"
---
# Derived Doc Capsules And Drift Audit

Created: 2026-04-16T04:38:00Z

## Summary

Add tiny derived capsules for large Brain-managed docs and audit them for drift so the compiler can use compact summaries by default while the full markdown docs remain the source of truth.

## Why It Matters

Brain already has hand-built summaries for a few core sources, but not a generalized capsule model or any explicit drift audit. Capsules only matter if they solve remaining packet pressure after budgets and session reuse land, and they should extend the existing summary-and-anchor model rather than create a second summary subsystem.

## Spec

- [[.brain/planning/specs/derived-doc-capsules-and-drift-audit.md]]

## Sources

- [[.brain/brainstorms/token-efficient-context-direction.md]]
- [[.brain/planning/specs/v1-base-contract-and-summary-anchors.md]]
- [[.brain/planning/specs/v1-context-packet-compiler.md]]
- [[.brain/planning/specs/release-install-and-update-flow.md]]

## Progress

- Spec approved.
- Story set created and reviewed against spec coverage.
- Execution order stays: capsule model and source coverage, deterministic generation, compiler preference, drift audit, then docs/skill follow-through.
- This epic should stay conditional on budgets plus reuse leaving meaningful document-cost pressure.
- A 2026-04-16 branch-code evaluation found that representative repeated compiles already shrink from roughly `1425-1431` human-estimated tokens to `281-287` as compact `reused` responses, and a clean fingerprint change still stayed near `283` human-estimated tokens as a compact `delta` response. Keep capsules gated until there is evidence the remaining first-turn full packet cost is hurting quality enough to justify the drift-audit complexity.
- Refined future direction: if this epic is revived later, the first operator-facing shape should be `capsules=off|auto|on`, default `off`, with `auto` making per-compile decisions from local fresh-packet telemetry instead of silently flipping a permanent global switch.
- Treat this epic as parked, not active roadmap work. Its value is as a constrained fallback if telemetry later shows repeated first-turn omission of the same high-signal docs.

## Notes

Keep full docs canonical. Capsules are derived compiler inputs with exact anchors and hashes, not a parallel truth system.
