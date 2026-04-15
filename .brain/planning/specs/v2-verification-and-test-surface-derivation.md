---
created: "2026-04-15T03:55:57Z"
epic: v2-verification-and-test-surface-derivation
project: brain
status: approved
title: V2 Verification And Test Surface Derivation Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V2 Verification And Test Surface Derivation Spec

Created: 2026-04-15T03:40:00Z

## Why

Brain's packet should answer not only what context matters next, but how the resulting change should be checked. Nearby tests and verification recipes are among the highest-value context items for implementation work.

## Problem

Current verification hints are still too incidental and uneven. Brain can see some nearby tests and required commands, but it does not yet derive a stable packet-friendly verification surface that combines repo structure, scripts, CI, and successful local patterns.

## Goals

- Derive nearby tests as structured packet candidates.
- Derive reusable verification recipes from the repo and observed successful commands.
- Improve packet usefulness for implementation work without overloading the default packet.
- Keep verification hints inspectable and grounded in observable sources.

## Non-Goals

- Predicting perfect verification for every task.
- Adding hosted or opaque test-selection systems.
- Replacing explicit policy docs with silent heuristics.
- Coupling verification guidance to a heavy plugin or workflow platform.

## Requirements

- Add a reusable verification-recipe representation for packet inclusion.
- Derive first-wave recipes from observable sources such as:
  - make targets
  - CI config
  - package scripts
  - recorded successful session commands when appropriate
- Add nearby-test derivation tied to changed files and normalized boundaries.
- Include verification hints in packet output with explicit reasons.
- Distinguish between strongly recommended verification and weaker nearby-test suggestions.
- Keep packet size bounded by selecting only the most relevant verification surfaces.

## UX / Flows

Implementation task flow:
1. User asks for a code change.
2. Brain compiles the packet.
3. Brain includes likely verification commands and nearby tests.
4. The packet explains why those commands or tests were included.

Ambiguous verification flow:
1. The repo has multiple possible verification routes.
2. Brain includes the strongest likely recipe and a smaller ambiguity note when alternatives are plausible.
3. Users can inspect the underlying sources when needed.

## Data / Interfaces

Suggested first-wave verification item fields:
- `label`
- `command`
- `source`
- `reason`
- `strength` (`strong` or `suggested`)

Suggested nearby-test fields:
- `path`
- `relation`
- `reason`

Recipe sources should remain visible so users can tell whether a hint came from CI, a make target, or observed successful session commands.

## Risks / Open Questions

- How much should observed successful session commands influence recipe selection before telemetry work exists?
- Should verification recipes be derived centrally or split between structural and live-work subsystems?
- How should Brain suppress redundant or overly broad test suggestions in large repos?

## Rollout

1. Add reusable verification-recipe and nearby-test item shapes.
2. Derive first-wave recipes and nearby-test surfaces from repo structure and existing signals.
3. Integrate the strongest verification surfaces into packet compilation.
4. Add diagnostics and tests that prove the hints are inspectable and bounded.

## Story Breakdown

- [ ] Derive Nearby Tests From Changed Files And Boundaries
- [ ] Extract Verification Recipes From Repo And Successful Command Sources
- [ ] Integrate Verification And Test Surfaces Into Compiled Packets

## Resources

- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/live-work-context.md]]
- [[.brain/planning/specs/structural-repo-context.md]]

## Notes

This epic should improve packet usefulness for real implementation work, not turn Brain into a full verification planner.
