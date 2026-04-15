---
created: "2026-04-15T03:55:57Z"
epic: v3-context-utility-analysis-surfaces
project: brain
status: approved
title: V3 Context Utility Analysis Surfaces Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V3 Context Utility Analysis Surfaces Spec

Created: 2026-04-15T03:40:00Z

## Why

As Brain starts logging packet usefulness, it needs direct inspection surfaces that let users see what was included, why it was included, what got expanded, and which items are trending toward signal or noise.

## Problem

A telemetry layer on its own would be hard to trust. Without human-facing analysis surfaces, Brain would have no clear way to explain packet rationale or show whether repeated inclusions are actually helping.

## Goals

- Add first-class inspection surfaces for packet rationale and utility.
- Make included versus expanded context visible.
- Surface likely signal and likely noise items from observed usage.
- Keep explanations rooted in recorded facts and inspectable heuristics.

## Non-Goals

- A full generic observability platform.
- Hiding explanations behind opaque confidence labels alone.
- Automatic tuning in this epic.
- Making utility stats mandatory for normal usage.

## Requirements

- Add `brain context explain` as a human-facing command for packet rationale.
- Add `brain context stats` as a human-facing command for utility summaries.
- Support inspection of:
  - why items were included
  - which items were later expanded
  - which items correlate with successful sessions
  - which items are repeatedly included but appear unused
- Keep explanations grounded in compiler reasons and recorded telemetry.
- Make it easy to inspect the latest packet and recent packet history.
- Update `docs/usage.md` and `skills/brain/SKILL.md` so the new explain and stats surfaces are teachable once they exist.

## UX / Flows

Explain latest packet flow:
1. User compiles context.
2. User runs `brain context explain --last`.
3. Brain shows included items, inclusion reasons, expansions, and downstream outcomes where available.

Stats flow:
1. User runs `brain context stats`.
2. Brain reports likely signal items, likely noise items, and repeated expansion patterns.
3. User uses that output to tune notes, summaries, or later ranking logic.

## Data / Interfaces

Suggested explain sections:
- `## Packet`
- `## Included Items`
- `## Why Included`
- `## Expanded Later`
- `## Downstream Outcomes`

Suggested stats sections:
- `## Top Signal`
- `## Top Noise`
- `## Frequently Expanded`
- `## Common Verification Links`

Suggested diagnostic fields:
- `include_count`
- `expand_count`
- `successful_verification_count`
- `durable_update_count`
- `likely_utility`

## Risks / Open Questions

- How much analysis should be exposed by default before the output becomes noisy?
- Should `context explain` target a packet hash, latest packet, or task summary first?
- How should Brain phrase "likely noise" without overstating weak telemetry?

## Rollout

1. Add packet explain surfaces backed by recorded packet and telemetry data.
2. Add utility summary surfaces for signal/noise inspection.
3. Keep the outputs inspectable and conservative in language.
4. Use this epic to validate whether the telemetry model is good enough for later ranking work.

## Story Breakdown

- [ ] Add Context Explain Surfaces For Packet Rationale
- [ ] Add Context Stats For Signal Noise And Expansion Patterns
- [ ] Add Conservative Utility Wording And Diagnostic Fields
- [ ] Teach Docs And The Brain Skill About Explain And Stats

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v3-context-packet-telemetry.md]]

## Notes

Users should be able to understand compiler behavior from these surfaces without needing to inspect raw storage directly.
