---
created: "2026-04-11T21:53:09Z"
epic: context-and-session-workflow
project: brain
status: approved
title: Context And Session Workflow Spec
type: spec
updated: "2026-04-13T06:23:52Z"
---
# Context And Session Workflow Spec

Created: 2026-04-11T21:53:09Z

## Why

Context and session enforcement should help an agent stay focused and leave the repo in a better state. Right now the repo contract is strong, but context loading is heavier than it needs to be and session closeout is stricter than it is helpful when memory updates are missing.

## Problem

Brain's current context model is effectively all-or-nothing: agents either read a broad set of files or risk missing something important. Session finish can also detect missing durable memory work without giving the user a first-class recovery path. The result is friction where Brain should be guiding the next step.

## Goals

- Introduce layered context loading so agents can start from a minimal contract and pull deeper context on demand.
- Make session closeout point users toward the right recovery action when durable memory updates are missing.
- Keep enforcement explicit and deterministic while reducing avoidable workflow friction.

## Non-Goals

- Background automatic context loading with opaque heuristics.
- Silent memory writes at session finish.
- A durable long-term store for temporary scratch memory.

## Requirements

- Add a context-loading interface that supports at least four levels: always-on identity/current-state, lightweight summaries, full context files, and deep search-driven context.
- Make the active context tiering obvious enough that agents can request more depth intentionally.
- When session finish sees meaningful repo change without memory updates, suggest `brain distill --session` or equivalent as the first recovery path.
- Reuse the same distillation primitives as the planning-memory workflow instead of inventing a separate closeout-only mechanism.
- Treat any session memory cache as optional, temporary, and explicitly non-durable until distilled.

## UX / Flows

Layered context flow:
1. Agent loads minimal startup context.
2. Agent requests the next level only when the task needs more detail.
3. Brain makes the transition from static context to search-driven deep context explicit.

Session closeout recovery flow:
1. User runs `brain session finish`.
2. Brain detects repo changes without acceptable memory updates.
3. Brain suggests a session-scoped distill flow instead of only returning a hard stop.
4. User reviews the proposed memory updates, accepts what matters, and retries closeout.

## Data / Interfaces

- Define context tiers in Brain-managed docs or config in a way that remains deterministic.
- Thread active session information into distill so closeout can scope the candidate updates.
- Keep any temporary session cache out of the accepted durable-memory globs unless explicitly promoted.

## Risks / Open Questions

- How much of layered context belongs in generated docs versus command output?
- Should session closeout suggest distill unconditionally, or only when certain note classes are missing?
- Does a temporary session cache help enough to justify another concept in the workflow?

## Rollout

1. Add layered context-loading primitives and documentation.
2. Add session closeout suggestions that reuse distill once manual distill exists.
3. Evaluate whether a temporary session cache is needed after those two steps land.

## Story Breakdown

- [ ] Add Layered Context Loading Levels
- [ ] Suggest Distillation During Session Closeout

## Resources

- [[.brain/brainstorms/mempalace-inspired-brain-improvements.md]]
- [[.brain/planning/stories/context-and-session-workflow-current-state-and-next-actions.md]]
- [[.brain/resources/references/agent-workflow.md]]

## Notes

The optional session-memory-cache idea stays out of the initial execution set until the lower-friction closeout path is proven useful.
