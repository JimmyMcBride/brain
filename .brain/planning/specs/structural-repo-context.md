---
created: "2026-04-13T22:00:56Z"
epic: structural-repo-context
project: brain
status: draft
title: Structural Repo Context Spec
type: spec
updated: "2026-04-13T22:02:40Z"
---
# Structural Repo Context Spec

Created: 2026-04-13T22:00:56Z

## Why

Brain currently has durable markdown context and deterministic generated docs, but it lacks a cheap structural layer that helps an agent orient around the repository before deeper retrieval. A lightweight repo map is the most practical next step.

## Problem

Without structural repo context, Brain has to infer too much from docs and note retrieval. That makes the system weaker at quickly identifying likely boundaries, entrypoints, test surfaces, and adjacent areas of code that matter for a task.

## Goals

- Add a compact derived structural layer that improves orientation and context selection.
- Keep the first wave language-agnostic and deterministic.
- Prefer useful repo boundaries and entrypoints over pretending Brain has deep parser-grade code understanding.
- Feed better structural signals into task-context assembly.

## Non-Goals

- Building a full semantic code graph.
- Shipping deep language-specific symbol analysis in the first wave.
- Replacing direct code reading with structural summaries.
- Expanding the canonical truth model beyond markdown.

## Requirements

- Derive structural repo context from the local workspace in a refreshable, disposable form.
- Cover at least repo tree summaries, important directories, entrypoints, config surfaces, test surfaces, and module or service boundaries where detectable.
- Add lightweight symbol or signature hints only where the signal is strong enough to stay reliable in a language-agnostic first pass.
- Keep the structural output compact enough to be included in task-context assembly without dominating the packet.
- Make the structural layer inspectable and debuggable rather than silently hidden inside ranking heuristics.

## UX / Flows

Structural orientation flow:
1. User asks Brain for task-relevant context.
2. Brain uses the structural layer to identify likely boundaries, entrypoints, tests, and adjacent code surfaces.
3. Brain includes the most relevant structural context in the assembled output.

Structural debugging flow:
1. A user wants to understand why Brain keeps pointing at certain parts of the repo.
2. Brain can expose the structural repo context directly enough to inspect what boundaries and entrypoints it believes exist.

## Data / Interfaces

- Treat structural repo context as derived local state, not durable project memory.
- The first-wave model should operate on workspace structure and lightweight code hints rather than language-specific parser outputs.
- Structural outputs should be consumable by task-context assembly and by lower-level debug surfaces when needed.
- Keep the representation generic enough to work across repos that are not Go-specific.

## Risks / Open Questions

- How much symbol-level detail can Brain safely expose without language-aware parsing?
- Which repo boundaries are stable enough to infer generically across project layouts?
- How should structural context degrade in repos with weak conventional structure?

## Rollout

1. Define the minimum viable structural layer and its inspection surface.
2. Add boundary, entrypoint, config, and test summaries.
3. Add lightweight symbol hints only where they improve task-context quality.
4. Tune the structural layer against real repositories before considering deeper code intelligence.

## Story Breakdown

- [ ] Define the minimum viable structural repo map Brain should derive.
- [ ] Add boundary, entrypoint, config, and test-surface summaries.
- [ ] Add modest symbol hints only where the signal is reliable enough to keep.

## Resources

- [[.brain/brainstorms/context-engine-v2.md]]
- [[.brain/resources/references/architecture-and-code-map.md]]
- [[.brain/resources/references/retrieval-and-indexing.md]]
- [[.brain/planning/specs/task-context-assembly.md]]

## Notes

The value of this epic is compression and orientation, not depth. If a detail cannot be derived reliably in a language-agnostic pass, leave it out of the first wave.
