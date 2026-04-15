---
created: "2026-04-15T03:55:58Z"
epic: v4-promotion-gate-and-durable-promotion-types
project: brain
status: approved
title: V4 Promotion Gate And Durable Promotion Types Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V4 Promotion Gate And Durable Promotion Types Spec

Created: 2026-04-15T03:40:00Z

## Why

Brain's compiler and session model will accumulate more ephemeral findings over time. The product needs an explicit gate that protects durable memory from transient scratch while still preserving valuable decisions, invariants, and recurring gotchas.

## Problem

Current closeout and distillation workflows can propose durable updates, but they do not yet define a strict compiler-era promotion model that says what is allowed to survive and what should die with the session.

## Goals

- Define a strict promotion model for compiler-era Brain.
- Classify what is promotable into durable memory.
- Prevent ephemeral reasoning and dead-end scratch from becoming durable by default.
- Keep promotion reviewable and explicit.

## Non-Goals

- Auto-writing broad session transcripts into durable memory.
- Preserving speculative reasoning as first-class memory.
- Replacing existing durable note editing with hidden automation.
- Turning promotion into a generic note dump path.

## Requirements

- Define first-wave promotable categories such as:
  - decisions
  - invariants
  - gotchas
  - verification recipes
  - boundary facts
  - unresolved follow-up items
- Define non-promotable default categories such as:
  - speculative reasoning
  - transient scratch notes
  - dead-end experiments unless proven recurring traps
- Add a promotion-gate representation that can classify proposed items before persistence.
- Keep promotion reviewable before writing durable notes.
- Make promotion rules visible enough that users can understand why something was or was not proposed.

## UX / Flows

Promotion review flow:
1. Session work surfaces candidate findings.
2. Brain classifies them against promotion rules.
3. Brain proposes only promotable items for durable capture.
4. User reviews and applies the durable update.

Non-promotable discard flow:
1. Session scratch accumulates during work.
2. Session closeout arrives.
3. Brain does not promote transient or speculative content by default.
4. The scratch dies with the session unless explicitly captured elsewhere.

## Data / Interfaces

Suggested promotion fields:
- `category`
- `source_packet_hash`
- `source_session_id`
- `proposed_target`
- `reason_promotable`
- `reason_rejected`
- `requires_review`

Suggested first-wave categories:
- `decision`
- `invariant`
- `gotcha`
- `verification_recipe`
- `boundary_fact`
- `follow_up`

## Risks / Open Questions

- How should Brain handle items that look promotable but lack enough confidence or specificity?
- Should recurring failed approaches ever become a first-class promotable category, or only when explicitly reviewed as traps?
- How should promotion categories map onto existing note types and docs without adding churn?

## Rollout

1. Define promotion categories and rejection rules.
2. Add gate logic for classifying candidate durable findings.
3. Keep the first wave review-first and conservative.
4. Integrate later closeout and distillation work with the new gate.

## Story Breakdown

- [ ] Define Promotable And Non-Promotable Categories For Compiler-Era Brain
- [ ] Add Promotion-Gate Classification And Diagnostics
- [ ] Integrate Promotion Gate With Durable Update Workflows

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/brainstorms/context-substrate-direction.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Notes

If this epic makes durable memory easier to spam, it has failed.
