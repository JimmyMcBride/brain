---
created: "2026-04-15T03:55:57Z"
epic: v3-utility-aware-context-ranking
project: brain
status: approved
title: V3 Utility-Aware Context Ranking Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V3 Utility-Aware Context Ranking Spec

Created: 2026-04-15T03:40:00Z

## Why

Once Brain can observe which packet items were repeatedly useful, it should be able to use that signal to improve future selection. This is the step where the compiler begins to benefit from local experience instead of only static scoring rules.

## Problem

Even with better structure and telemetry, packet selection can remain too flat if Brain treats all matching items as equally valuable. It needs a cautious way to reward repeatedly useful context and de-emphasize repeated noise.

## Goals

- Add utility-aware scoring to packet selection.
- Reward context items that repeatedly correlate with successful outcomes or meaningful expansions.
- Penalize repeated inclusions that show little downstream impact.
- Keep the ranking model inspectable and bounded.

## Non-Goals

- Opaque machine-learning ranking systems.
- Remote training or cross-repo sharing.
- Letting utility scores override obvious task or boundary mismatches.
- Treating weak telemetry as strong truth.

## Requirements

- Introduce utility-aware scoring inputs such as:
  - repeated successful verification linkage
  - repeated durable update linkage
  - repeated expansion after inclusion
  - repeated inclusion without expansion or downstream effect
- Blend utility with deterministic compiler signals rather than replacing them.
- Keep lexical/task match, boundary overlap, and workflow-required items as first-class ranking factors.
- Surface utility-aware reasons in explain output.
- Ensure the same repo state and telemetry state produce the same ranking result.

## UX / Flows

Useful-context reuse flow:
1. Similar tasks repeatedly compile packets.
2. Certain items are repeatedly expanded or tied to successful work.
3. Brain raises those items slightly in future ranking for comparable tasks.

Noise suppression flow:
1. Some items are repeatedly included but rarely expanded or connected to good outcomes.
2. Brain lowers those items modestly in future ranking.
3. Explain output makes the utility adjustment visible.

## Data / Interfaces

Suggested first-wave blended scoring model:
- `task_match`
- `boundary_overlap`
- `changed_file_overlap`
- `workflow_required_bonus`
- `utility_bonus`
- `noise_penalty`
- `stale_penalty`
- `token_cost_penalty`

Explain diagnostics should support reasons such as:
- `repeatedly expanded in similar recent sessions`
- `often included with successful verification`
- `repeated inclusion with low downstream use`

## Risks / Open Questions

- How much recent history should influence ranking before utility becomes unstable?
- What is the minimum telemetry volume required before utility adjustments should activate?
- How should Brain prevent one noisy historical run from distorting future packets?

## Rollout

1. Add utility and noise scoring inputs.
2. Blend them conservatively into the existing deterministic selector.
3. Expose utility-aware reasons in explain output.
4. Tune activation thresholds so ranking changes only after enough evidence exists.

## Story Breakdown

- [ ] Add Conservative Utility And Noise Scoring Inputs
- [ ] Blend Utility With Deterministic Task Boundary And Workflow Signals
- [ ] Surface Utility-Aware Reasons And Threshold Diagnostics

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/v3-context-packet-telemetry.md]]
- [[.brain/planning/specs/v3-context-utility-analysis-surfaces.md]]

## Notes

The safest default is to make utility a tie-breaker or moderate bias first, not a dominant selector.
