---
created: "2026-04-13T22:00:47Z"
epic: task-context-assembly
project: brain
status: draft
title: Task Context Assembly Spec
type: spec
updated: "2026-04-13T22:02:40Z"
---
# Task Context Assembly Spec

Created: 2026-04-13T22:00:47Z

## Why

Brain's strongest properties are trust, inspectability, and local-first workflow control. The next meaningful gain is not more memory. It is assembling better task-specific context from the trustworthy sources Brain already owns or can derive locally.

## Problem

Today Brain can load deterministic context files and retrieve durable markdown notes, but it still leaves too much work to the agent when deciding what context matters for a specific task. The product needs a more intentional task-context assembly flow that can combine multiple source types and explain why each one was selected.

## Goals

- Improve Brain's answer to: what context does this task need right now?
- Move from mostly note-centric retrieval toward typed local context assembly.
- Keep selected context transparent enough that users can see why it was chosen and what was left out.
- Reuse existing command surfaces where practical instead of adding a new flagship command too early.

## Non-Goals

- Replacing markdown as canonical truth.
- Turning Brain into a hosted or opaque memory system.
- Designing a broad relationship-graph platform in the first wave.
- Committing the first wave to a brand-new top-level packet command before the workflow proves necessary.

## Requirements

- Define a task-context assembly flow that can intentionally combine canonical markdown, generated project docs, structural repo context, live work signals, and policy/workflow guidance.
- Group selected context by type so the output makes it obvious what kind of source is being included.
- Include selection rationale for each source or source group.
- Surface nearby but omitted context so the agent can see obvious alternatives or ambiguities.
- Keep the output compact enough to act as an agent handoff rather than another broad document dump.
- Prefer evolution of `brain context load` and `brain search` before introducing a dedicated new top-level command.

## UX / Flows

Task-context assembly during active work:
1. User starts or validates a session.
2. User requests task-relevant context for the current task or a provided query.
3. Brain combines the best local sources by type.
4. Brain returns a visible bundle that includes selected context, rationale, omitted-nearby context, and ambiguity notes.

Task-context assembly without an active session:
1. User provides a task or query explicitly.
2. Brain assembles the same typed context bundle using repo state plus the explicit task text.
3. Brain remains transparent about weaker confidence when active-work signals are missing.

## Data / Interfaces

- Treat the assembled task-context bundle as a read-only derived output.
- Preserve existing deterministic context files and search results as lower-level primitives that feed the assembled output.
- Keep the interface markdown-native and JSON-friendly so the same assembled result is usable in terminal output, tests, and agent integrations.
- Model source groups at least at the level of markdown truth, generated docs, structural repo context, live work signals, and policy/workflow guidance.

## Risks / Open Questions

- How much can existing `context load` and `search` surfaces stretch before a dedicated command becomes cleaner?
- How much omitted-context detail is useful before the output becomes noisy?
- Which confidence or ambiguity markers are actually helpful in real use instead of decorative?

## Rollout

1. Define the assembled-output contract and rationale model.
2. Reuse current context and search primitives to prove the workflow.
3. Add richer typed inputs from the structural and live-work epics.
4. Re-evaluate whether a dedicated context-packet command is justified after the assembly flow exists.

## Story Breakdown

- [ ] Define the task-context output contract and explanation model.
- [ ] Extend current context/search flows to emit typed assembled context.
- [ ] Add omitted-nearby and ambiguity reporting without overwhelming the output.

## Resources

- [[.brain/brainstorms/context-engine-v2.md]]
- [[.brain/planning/specs/retrieval-and-index-lifecycle.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]
- [[.brain/resources/references/skills-and-context-engineering.md]]

## Notes

This epic is the product center of the initiative. The other new epics exist mainly to improve the quality of this assembled output.
