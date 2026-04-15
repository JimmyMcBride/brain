---
created: "2026-04-15T03:55:57Z"
epic: v1-base-contract-and-summary-anchors
project: brain
status: approved
title: V1 Base Contract And Summary Anchors Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V1 Base Contract And Summary Anchors Spec

Created: 2026-04-15T03:40:00Z

## Why

Brain is moving toward a deterministic context compiler. That requires a small stable base contract plus reusable summary-and-anchor forms for the major context sources Brain already owns.

## Problem

Current context surfaces still lean too heavily on loading full documents or large generated bundles. That makes context selection expensive, weakens packet assembly, and blurs the distinction between default boot context and deeper expansion.

## Goals

- Define a reusable `ContextItem` shape for compiler-oriented context selection.
- Establish a tiny always-on base contract derived from the project contract and generated context.
- Represent major context sources as lossy summaries with exact expansion anchors.
- Make the base contract small enough to be safely included by default.
- Preserve inspectability by keeping every summary tied to a precise source anchor.

## Non-Goals

- Shipping the full packet compiler command in this epic.
- Adding telemetry-based ranking or adaptive selection.
- Expanding every Brain note into rich structured metadata immediately.
- Replacing markdown as the canonical durable memory layer.

## Requirements

- Add a unified `ContextItem` model for reusable context selection.
- Add an explicit anchor representation that can point to exact source paths and sections.
- Define the base contract item set for the current product direction:
  - boot summary
  - workflow contract
  - memory/update rules
  - architecture summary
  - verification summary
- Generate summary-plus-anchor forms for the major Brain-owned context sources used by startup and packet assembly.
- Keep base-contract summaries intentionally tiny and safe for default include.
- Preserve deterministic generation so the same repo state produces the same base-contract items.
- Make the initial item set available to later packet assembly without requiring a full index system.

## UX / Flows

Default compiler boot flow:
1. Brain resolves the current project.
2. Brain loads the compact base-contract items.
3. Brain uses those items as the smallest always-available contract before task-specific selection begins.

Source expansion flow:
1. A summary item is selected into a packet.
2. The packet exposes the exact anchor for deeper inspection.
3. Later commands can expand from the anchor without guessing where the summary came from.

## Data / Interfaces

Suggested first-wave item fields:
- `id`
- `kind`
- `title`
- `summary`
- `anchor_path`
- `anchor_section`
- `boundaries`
- `files`
- `source_hash`
- `expansion_cost`

First-wave item kinds in this epic:
- `base_contract`
- `durable_note`
- `generated_context`
- `workflow_rule`
- `verification_recipe`

Base-contract item constraints:
- each item should be short enough to function as a summary, not a mini-document
- each item must carry a deterministic anchor
- item generation must be inspectable and testable

## Risks / Open Questions

- How small can the base contract get before it becomes too lossy to be useful?
- Should verification guidance be one base item or a small clustered group?
- Which generated context docs are stable enough to summarize directly versus requiring a dedicated reduction pass?

## Rollout

1. Add `ContextItem` and anchor primitives.
2. Generate the first-wave base-contract items.
3. Add summary-plus-anchor generation for major Brain-owned context sources.
4. Expose these items to the later packet compiler work.

## Story Breakdown

- [ ] Add Compiler Context Item And Anchor Types
- [ ] Build Tiny Base Contract Extraction
- [ ] Generate Summary And Anchor Forms For Major Context Sources

## Resources

- [[.brain/brainstorms/context-substrate-direction.md]]
- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/task-context-assembly.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]

## Notes

This epic should make Brain's default context smaller, not larger. If the first implementation increases startup context size, it is moving in the wrong direction.
