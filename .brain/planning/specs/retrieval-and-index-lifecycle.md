---
created: "2026-04-11T21:53:09Z"
epic: retrieval-and-index-lifecycle
project: brain
status: approved
title: Retrieval And Index Lifecycle Spec
type: spec
updated: "2026-04-13T06:23:52Z"
---
# Retrieval And Index Lifecycle Spec

Created: 2026-04-11T21:53:09Z

## Why

Brain's retrieval layer is already useful, but the next gain is not more storage. It is better ranking and better handoff into agent work. Search should answer what matters right now and make the answer easier to use.

## Problem

Current hybrid search is query-matching oriented. It does not yet strongly account for recency, note importance, or the active work context from the current session, epic, or spec. Search results also stop at retrieval, forcing the agent to manually convert ranked matches into a usable context block.

## Goals

- Improve ranking quality with recency, note-type weighting, and active-work-context signals.
- Add a search-to-context bridge that returns agent-ready injected context, not just ranked hits.
- Keep retrieval project-local and markdown-backed while improving the quality of what is returned.

## Non-Goals

- Turning Brain into a hosted semantic retrieval system.
- Replacing explicit note structure with opaque embedding-only ranking.
- Introducing filesystem-like room or wing abstractions on top of the repo.

## Requirements

- Add recency-sensitive ranking so recently updated notes can receive a measured boost.
- Weight note types so durable decisions, specs, and change notes can outrank brainstorm noise when relevance is otherwise close.
- Use active session context, current epic or spec, or nearby planning metadata to bias results toward the work in front of the agent.
- Add `brain search --inject` or equivalent output that includes both ranked results and a preformatted relevant-context block.
- Keep ranking explainability good enough that `brain search --explain` can still make the score story understandable.

## UX / Flows

Search with injection:
1. Agent runs `brain search --inject "query"`.
2. Brain ranks results with lexical, semantic, recency, type, and active-context signals.
3. Output includes both the normal ranked matches and a compact `Relevant Context` block ready to reuse.

Search during active work:
1. User starts or validates a session.
2. Brain knows the active project task and nearby planning notes.
3. Retrieval quietly boosts the notes tied to the active work while keeping the result explainable.

## Data / Interfaces

- Extend ranking inputs to include note timestamps, note types, and active planning context.
- Define an injected-context output format that stays markdown-native and easy to paste into agent prompts.
- Preserve explain output so retrieval tuning remains debuggable.

## Risks / Open Questions

- How aggressive should recency and type weighting be before search starts hiding older but critical notes?
- Which active-context signals are reliable enough to boost by default?
- Should injected context be a new flag, a new format mode, or part of broader context loading?

## Rollout

1. Add ranking features behind explainable scoring changes.
2. Land injected-context output once the ranking results are trustworthy.
3. Tune weights against real project notes and session flows.

## Story Breakdown

- [ ] Improve Retrieval Ranking With Recency And Active Context
- [ ] Turn Search Results Into Agent-Ready Context Blocks

## Resources

- [[.brain/brainstorms/mempalace-inspired-brain-improvements.md]]
- [[.brain/resources/references/retrieval-and-indexing.md]]
- [[.brain/planning/stories/retrieval-and-index-lifecycle-current-state-and-next-actions.md]]

## Notes

This spec improves how Brain chooses and packages context. It does not change the core local-first storage model.
