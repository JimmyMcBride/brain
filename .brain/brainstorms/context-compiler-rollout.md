---
brainstorm_status: active
created: "2026-04-15T03:38:01Z"
idea_count: 0
project: brain
title: Context Compiler Rollout
type: brainstorm
updated: "2026-04-15T04:01:19Z"
---
# Brainstorm: Context Compiler Rollout

Started: 2026-04-15T03:05:18Z

## Focus Question

How should Brain roll out the deterministic context-compiler architecture in a way that sharpens memory and context quality quickly without turning the rewrite into a large speculative systems project?

## Ideas

- Build this in four versions.
  Four versions is enough to separate the valuable core packet model from the later adaptive and promotion systems.
- Make `v1` do almost all of the identity shift.
  If `v1` can emit a compact justified packet for the current task, Brain already feels much closer to the intended product.
- Keep the base contract tiny.
  `AGENTS.md` and generated context should get smaller and more bootloader-like, not larger.
- Make summaries first-class from the start.
  Every reusable context source should move toward summary plus anchor, even before telemetry and advanced scoring exist.
- Prefer deterministic signals before adaptive ones.
  Changed files, boundaries, nearby tests, note links, and workflow requirements should drive early selection before any “learning” layer exists.
- Preserve compatibility while changing the center of gravity.
  Existing `context load`, `context live`, and `context assemble` surfaces can remain for a while, but `context compile` should become the new mental model underneath them.
- Delay expensive system growth until the packet model proves useful.
  `contextindex`, `contexttelemetry`, and promotion flows should arrive only after the core packet format proves it helps real tasks.

## Rollout Summary

- `v1`: deterministic packet compiler
- `v2`: boundary-aware code context
- `v3`: usefulness telemetry and tuning
- `v4`: promotion pipeline and mental-model cleanup

## Version Plan

### `v1`: Deterministic Packet Compiler

Goal:
Ship the first version of Brain as a context compiler rather than a document loader.

Scope:
- introduce a unified `ContextItem` model
- build tiny base-contract items from the current project contract and generated context
- add summary-plus-anchor handling for major durable context sources
- add `brain context compile`
- assemble a packet from:
  - base contract
  - changed files
  - touched boundaries
  - nearby tests
  - top durable note summaries
- attach inclusion reasons to every packet item
- record packet composition in session state

Acceptance criteria:
- `brain context compile --task "..." --json` returns a structured task packet
- packet output includes task framing, boundaries, files, tests, summaries, verification hints, and inclusion reasons
- the default packet is summary-first rather than full-document-first
- packet generation is deterministic for the same repo state and task input
- session state records which context items were included in the packet
- no telemetry-based reranking is required yet

Likely package/file focus:
- `internal/projectcontext/`
- `internal/taskcontext/`
- `internal/livecontext/`
- `internal/session/`
- `cmd/context_*.go` for `context compile`

Risks:
- trying to redesign every existing context command at once
- overcomplicating `ContextItem` before it has real callers

### `v2`: Boundary-Aware Code Context

Goal:
Make selection materially sharper by grounding packets in repo structure instead of only note search and task text.

Scope:
- add a repo boundary map
- add file-to-boundary and package/test adjacency data
- link durable notes to boundaries and affected files where possible
- improve candidate generation with boundary overlap and changed-file pressure
- improve verification hints from nearby test ownership and known recipes

Acceptance criteria:
- packets identify likely affected boundaries more accurately than `v1`
- packets consistently surface nearby tests when code changes are implied
- durable note summaries can be selected because of boundary linkage, not only lexical matches
- inclusion reasons can point to boundary overlap or changed-file overlap explicitly
- packet quality improves for multi-package tasks without large token growth

Likely package/file focus:
- `internal/livecontext/`
- `internal/projectcontext/`
- `internal/search/`
- possible new boundary-model files under `internal/`

Risks:
- boundary extraction becoming language- or repo-shape-specific too early
- building a graph system that is more complex than current repo needs justify

### `v3`: Usefulness Telemetry And Tuning

Goal:
Teach Brain which included context actually helps, using real session outcomes instead of guesswork.

Scope:
- record included item IDs and expanded item IDs
- link packet composition to verification outcome and durable note updates
- add utility and noise signals for context items
- add `brain context explain` and `brain context stats`
- begin simple reranking using observed utility

Acceptance criteria:
- Brain stores packet composition and expansion events locally
- Brain can report which context items are most often included, expanded, and associated with successful sessions
- `brain context explain` can show why packet items were included and whether they were later expanded or useful
- `brain context stats` can identify likely signal and likely noise items
- initial reranking remains inspectable and deterministic enough to debug

Likely package/file focus:
- `internal/session/`
- `internal/search/`
- possible new `internal/contexttelemetry/`
- `cmd/context_explain.go`
- `cmd/context_stats.go`

Risks:
- usefulness heuristics being treated as truth too early
- logging too much ephemeral detail and polluting local state

### `v4`: Promotion Pipeline And Mental-Model Cleanup

Goal:
Finish the architectural shift by separating ephemeral scratch from durable memory and retiring the old static-level mental model.

Scope:
- add a strict promotion pathway for moving ephemeral findings into durable memory
- classify what is promotable versus what should die with the session
- add session-close suggestions for promotable decisions, gotchas, verification recipes, and follow-ups
- retire or demote the old “context levels” framing in docs and CLI UX
- make `context compile` the primary context surface, with older commands as wrappers or compatibility layers where needed

Acceptance criteria:
- ephemeral state does not become durable memory by default
- promotable items are clearly typed and reviewable before persistence
- session close can suggest durable promotions without forcing transcript-like dumps
- docs and user guidance reflect the context-compiler model instead of static context bundles
- Brain’s primary context story becomes “compile the smallest useful packet” rather than “choose a load level”

Likely package/file focus:
- `internal/session/`
- `internal/projectcontext/`
- `internal/taskcontext/`
- `cmd/`
- `.brain/context/*`
- `docs/usage.md`, `docs/architecture.md`, `skills/brain/SKILL.md`

Risks:
- trying to auto-promote too much
- leaving both old and new mental models equally prominent for too long


## Epic Breakdown

### `v1`
- [[.brain/planning/epics/v1-base-contract-and-summary-anchors.md]]
- [[.brain/planning/epics/v1-context-packet-compiler.md]]
- [[.brain/planning/epics/v1-packet-provenance-and-session-recording.md]]

### `v2`
- [[.brain/planning/epics/v2-repo-boundary-graph.md]]
- [[.brain/planning/epics/v2-boundary-aware-context-selection.md]]
- [[.brain/planning/epics/v2-verification-and-test-surface-derivation.md]]

### `v3`
- [[.brain/planning/epics/v3-context-packet-telemetry.md]]
- [[.brain/planning/epics/v3-context-utility-analysis-surfaces.md]]
- [[.brain/planning/epics/v3-utility-aware-context-ranking.md]]

### `v4`
- [[.brain/planning/epics/v4-promotion-gate-and-durable-promotion-types.md]]
- [[.brain/planning/epics/v4-session-closeout-promotion-suggestions.md]]
- [[.brain/planning/epics/v4-context-compiler-ux-migration.md]]

## Recommended First Slice

If implementation starts now, the narrow first slice should be:

1. add `ContextItem`
2. add summaries plus anchors for major context sources
3. add `brain context compile`
4. compile a packet from:
   - base contract
   - changed files
   - touched boundaries
   - nearby tests
   - top durable note summaries
5. attach inclusion reasons
6. record packet composition in session state

This is the first meaningful product leap without needing a full rewrite.

## File-Level Starting Map

`internal/projectcontext/`
- build base-contract items
- generate summary-plus-anchor forms for project contract and generated context sources

`internal/taskcontext/`
- move from concatenation toward candidate gathering and packet assembly

`internal/livecontext/`
- expose normalized changed files, boundaries, nearby tests, and verification hints for packet compilation

`internal/session/`
- persist packet composition metadata for the active session

`cmd/`
- add `brain context compile`
- later add `context explain`, `context expand`, and `context stats`

## Out Of Scope For `v1`

- embedding-heavy ranking everywhere
- adaptive reranking from telemetry
- a large standalone context index package if current packages can carry the first slice
- complex promotion workflows
- rewriting all existing context commands immediately

## North Star

A strong Brain packet should answer:
- what am I trying to do?
- what boundaries are most likely involved?
- what is the minimum trustworthy context needed next?
- why was each included?
- how should I verify the result?

## Related

- Substrate direction: [[.brain/brainstorms/context-substrate-direction.md]]
- Current repo docs: [[docs/usage.md]]
- Current repo docs: [[docs/architecture.md]]
- Current repo docs: [[docs/skills.md]]

## Raw Notes

- This rollout replaces the broader planning/plugin direction with a deeper investment in memory and context quality.
- `v1` should be the point where Brain starts to feel like a deterministic context compiler instead of a document loader.
- `v2` is the point where repo structure should materially improve context sharpness.
- `v3` is when Brain starts learning from actual usefulness rather than assumed relevance.
- `v4` is where the product fully cleans up the old mental model and gains a disciplined promotion path.
- The biggest product mistake would be trying to land telemetry, learning, promotion, and graph sophistication before the basic packet format proves useful.
