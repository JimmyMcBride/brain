---
created: "2026-04-13T22:00:56Z"
epic: live-work-context
project: brain
status: draft
title: Live Work Context Spec
type: spec
updated: "2026-04-13T22:02:40Z"
---
# Live Work Context Spec

Created: 2026-04-13T22:00:56Z

## Why

The right context is not only about what is true in the repo. It is also about what is active now. Brain already has sessions, git-aware closeout, verification history, and planning artifacts, but it does not yet use those signals deeply enough to improve task focus.

## Problem

Brain currently has limited awareness of the specific work in progress beyond task text and note ranking. That leaves it weaker than it should be at choosing nearby tests, touched files, relevant docs, and policy guidance for the current task.

## Goals

- Add practical live-work signals that improve task focus without changing Brain's canonical truth model.
- Make active sessions, current diff, touched files, nearby tests, recent verification results, and planning proximity usable as typed derived context.
- Improve task-context assembly and retrieval quality with these signals while keeping the result explainable.
- Stop short of introducing a broad relationship-graph platform in the first wave.

## Non-Goals

- Building a repo-wide relationship graph as the first implementation.
- Storing git history or session telemetry as new durable memory by default.
- Replacing explicit policy or workflow docs with silent heuristics.
- Treating live-work context as stable truth when it is only current-state signal.

## Requirements

- Derive live-work context from active session state when available.
- Include current git diff or touched-file awareness as a first-class signal.
- Include nearby tests, recent verification commands and outcomes, and relevant planning artifact proximity where available.
- Include applicable workflow or policy guidance when the touched area or task suggests it matters.
- Keep the use of live-work context inspectable so a user can understand why Brain emphasized certain files, tests, docs, or rules.

## UX / Flows

Active-session task flow:
1. User starts or validates a session for a task.
2. Brain derives live-work context from the task, current repo state, and verification history.
3. Brain uses those signals to improve context assembly and retrieval around what is currently changing.

No-session fallback flow:
1. User requests task context without an active session.
2. Brain falls back to explicit query text plus git/workspace state.
3. Brain reports weaker confidence when important session signals are unavailable.

## Data / Interfaces

- Treat live-work signals as derived current-state context, not durable notes.
- Keep the interface compatible with existing session and search primitives.
- Model at least active task, changed files, nearby tests, verification outcomes, planning proximity, and applicable rule/policy guidance.
- Make the derived live-work signal set available to task-context assembly as a typed input rather than a hidden ranking-only side effect.

## Risks / Open Questions

- Which git or workspace signals are stable and cheap enough to use by default?
- How much verification history is helpful before the signal becomes noisy?
- When should policy or workflow guidance be selected automatically versus only suggested as nearby context?

## Rollout

1. Define the minimum live-work signal set.
2. Add active task, changed-file, and nearby-test awareness first.
3. Add verification and planning proximity signals next.
4. Revisit whether a broader relationship layer is still needed after these practical signals land.

## Story Breakdown

- [ ] Define the minimum live-work signal set Brain should derive.
- [ ] Add active task, changed-file, and nearby-test awareness to task-context assembly.
- [ ] Add verification, planning, and policy proximity signals with clear rationale output.

## Resources

- [[.brain/brainstorms/context-engine-v2.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]
- [[.brain/planning/specs/task-context-assembly.md]]
- [[.brain/resources/references/agent-workflow.md]]

## Notes

This epic should improve focus around real work in progress. If later evidence shows that practical live-work signals are still not enough, that is the point to consider a separate graph-oriented follow-on.
