---
brainstorm_status: active
created: "2026-04-13T06:23:51Z"
idea_count: 0
project: brain
title: MemPalace-Inspired Brain Improvements
type: brainstorm
updated: "2026-04-13T06:23:51Z"
---
# Brainstorm: MemPalace-Inspired Brain Improvements

Started: 2026-04-13T06:27:00Z

## Focus Question

Which MemPalace-style ideas are actually worth borrowing for Brain without breaking Brain's explicit, curated-memory philosophy?

## Ideas

- Optional session-aware memory distillation can recover decisions, tradeoffs, bugs, and discoveries that currently disappear when nobody writes them down.
- Distillation should propose note updates, not silently write them. Brain keeps human or agent approval in the loop.
- Layered context loading can split startup context into L0 identity/current-state, L1 workflow and architecture summary, L2 full context files, and L3 search-driven deep context.
- Retrieval ranking should favor what matters now, not just literal query overlap. Recency, note type, and active epic or spec context should affect ordering.
- Search should bridge directly into usage with an inject mode that returns a preformatted context block an agent can paste into work.
- Brain needs first-class decision notes so the system keeps why a choice was made, not just the final choice.
- Session closeout should help the user recover missing memory work by suggesting a session-scoped distill path when repo state changed but durable notes did not.
- A lightweight session memory cache is interesting only as an optional temporary layer. It should stay clearly separate from durable memory until distilled.

## Related

- [[.brain/planning/epics/planning-and-brainstorming-ux.md]]
- [[.brain/planning/epics/retrieval-and-index-lifecycle.md]]
- [[.brain/planning/epics/context-and-session-workflow.md]]

## Raw Notes

High-value changes from the brainstorm:

1. Memory capture: add an optional `brain distill` flow that inspects recent session activity, command runs, and git diff, then proposes durable note updates.
2. Context efficiency: add layered context loading so agents can start small and pull deeper context only when needed.
3. Retrieval quality: improve ranking with recency, note-type weighting, and active-work-context bias.
4. Reasoning preservation: add a first-class decision note type and template.
5. Workflow smoothing: integrate distillation suggestions into session closeout instead of only blocking on missing memory updates.
6. Retrieval-to-usage bridge: add `brain search --inject` or equivalent agent-ready context output.

Non-goals from the same brainstorm:

- Do not store everything verbatim.
- Do not introduce spatial memory abstractions that duplicate the filesystem.
- Do not lean harder on embeddings at the cost of explicit project structure.
