---
brainstorm_status: active
created: "2026-04-16T04:35:00Z"
idea_count: "8"
project: brain
title: Token-Efficient Context Direction
type: brainstorm
updated: "2026-04-16T02:41:00Z"
---
# Brainstorm: Token-Efficient Context Direction

Started: 2026-04-16T04:35:00Z

## Focus Question

How should Brain capture most of the token savings from layered AI-facing documentation without turning into a Cursor-style always-injected rule system?

## Ideas

- Keep Brain task-first, not file-glob-first. Selection should still center on task text, touched boundaries, changed files, nearby tests, policy, and session state.
- Add hard packet budgets so `brain context compile` selects under an explicit token target instead of only fixed item-count caps.
- Reuse compiled packets across turns inside a session when the task and relevant repo state have not changed, then emit deltas when they have.
- Derive compact doc capsules from Brain-managed source docs so the compiler can pick tiny summaries first and only expand the full source when needed.
- Treat capsules as derived artifacts with exact anchors and source hashes, not as separate truth.
- Add drift auditing so a source-doc change without a matching capsule refresh is visible instead of silently degrading packet quality.
- Avoid always-injected `.mdc`-style rule packs as the main Brain model. They help in short editor conversations but become repeated prompt tax in long sessions.
- The highest-leverage path is budgeted compile + packet reuse first, then doc capsules and drift audit.

## Related

- [[.brain/brainstorms/context-substrate-direction.md]]
- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v1-context-packet-compiler.md]]
- [[.brain/planning/specs/v2-boundary-aware-context-selection.md]]
- [[.brain/planning/specs/v3-context-packet-telemetry.md]]

## Raw Notes

20/80 recommendation locked for this branch:

1. Add hard packet budgets to the context compiler.
2. Reuse compiled packets across turns inside a session and only send compact reuse or delta outputs when that actually reduces repeated packet weight.
3. Add derived doc capsules and drift audit only if budgeted compile plus reuse still leave meaningful document-cost pressure.

Important constraints:

- Brain should not become a general rule engine.
- Brain should not re-inject the same short rule packs every turn.
- Brain should keep durable truth in human-readable markdown and treat capsules as derived compiler inputs.
- Packet reuse is only a real win if unchanged packet bodies are not re-emitted wholesale.
- Capsules should extend the existing summary-and-anchor path, not create a parallel summary system.
