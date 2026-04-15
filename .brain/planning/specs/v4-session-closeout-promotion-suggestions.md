---
created: "2026-04-15T03:55:58Z"
epic: v4-session-closeout-promotion-suggestions
project: brain
status: approved
title: V4 Session Closeout Promotion Suggestions Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V4 Session Closeout Promotion Suggestions Spec

Created: 2026-04-15T03:40:00Z

## Why

Once Brain has a promotion gate, the closeout flow should help users preserve the important results of packet-driven work. That closes the loop between context compilation, execution, verification, and durable memory.

## Problem

Current session closeout can enforce durable updates and distillation, but it does not yet use compiler-era packet history and promotion categories to suggest the right durable follow-through at the right time.

## Goals

- Add closeout-time promotion suggestions grounded in the new promotion model.
- Suggest only the most valuable durable follow-through from the session.
- Connect packet history, verification outcomes, and promotion categories.
- Keep suggestions review-first and conservative.

## Non-Goals

- Auto-writing durable memory at closeout.
- Forcing users through broad promotion prompts for every session.
- Preserving raw packet history as durable notes.
- Replacing explicit note editing or distillation review.

## Requirements

- Use session closeout as a suggestion point for promotable findings.
- Connect closeout suggestions to:
  - packet history
  - verification outcomes
  - durable updates already written
  - unresolved follow-up items where present
- Avoid suggesting promotions for non-promotable or weakly evidenced items.
- Keep closeout output actionable and bounded.
- Integrate with existing session-finish and distillation workflows rather than replacing them.

## UX / Flows

Clean closeout flow:
1. Session work completes successfully.
2. Brain inspects packet history and promotion candidates.
3. Brain suggests a small set of durable follow-through items if warranted.
4. User reviews and applies updates before final closeout.

No-suggestion flow:
1. Session work is trivial or already fully reflected in durable notes.
2. Brain detects no meaningful promotable findings.
3. Closeout proceeds without noisy extra prompts.

## Data / Interfaces

Suggested closeout suggestion fields:
- `category`
- `summary`
- `suggested_target`
- `supporting_packet_hashes`
- `supporting_verification`
- `reason_suggested`

Suggested suggestion sources:
- successful packet-driven work with meaningful verification
- repeated packet items tied to a decision or gotcha
- unresolved blockers or follow-ups left at closeout time

## Risks / Open Questions

- How should Brain phrase closeout suggestions so they feel helpful instead of nagging?
- Should closeout suggestions always route through `brain distill --session`, or can some go through lighter-weight review flows?
- How should Brain suppress duplicate suggestions across repeated sessions on the same task?

## Rollout

1. Add promotion-aware closeout suggestion logic.
2. Integrate packet history and verification support into closeout suggestions.
3. Keep the first-wave UX conservative and easy to skip when nothing useful exists.
4. Reuse existing durable update flows wherever possible.

## Story Breakdown

- [ ] Add Closeout-Time Promotion Suggestion Logic
- [ ] Connect Closeout Suggestions To Packet History Verification And Durable State
- [ ] Tune Closeout Suggestion UX And Suppression Rules

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v4-promotion-gate-and-durable-promotion-types.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Notes

A silent no-suggestion closeout is often the correct behavior.
