---
created: "2026-04-13T22:00:47Z"
epic: task-context-assembly
project: brain
status: approved
title: Task Context Assembly Spec
type: spec
updated: "2026-04-13T22:16:24Z"
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
- Reuse the existing `context` surface instead of adding a new top-level command.

## Non-Goals

- Replacing markdown as canonical truth.
- Turning Brain into a hosted or opaque memory system.
- Designing a broad relationship-graph platform in the first wave.
- Defining the structural derivation or live-work derivation logic that belongs to later epics.

## Requirements

- Add `brain context assemble` as the task-focused context-packet interface under the existing `context` command group.
- Support `brain context assemble --task "<task>"` and `brain context assemble` using the active session task.
- Return a compact task-context packet grouped by typed source categories.
- Include a rationale for each selected source or source group.
- Include ambiguity notes when Brain is uncertain or missing strong signals.
- Support an explicit explain mode that expands the packet with richer rationale and omitted-nearby context.
- Keep the first implementation compatible with current Brain primitives by building on existing context-loading and search behavior.
- Lock the full packet schema now, even if some source groups remain empty until later epics land.

## UX / Flows

Task-context assembly with explicit task:
1. User runs `brain context assemble --task "tighten auth flow"`.
2. Brain resolves the task from the flag.
3. Brain assembles a compact packet from typed local sources.
4. Brain returns the selected context, grouped by type, plus ambiguity notes when needed.

Task-context assembly with active session:
1. User starts or validates a session.
2. User runs `brain context assemble`.
3. Brain uses the active session task as the task input.
4. Brain returns the same typed packet and marks the task source as `session`.

Explain mode:
1. User runs `brain context assemble --explain`.
2. Brain returns the normal packet plus detailed selection rationale, omitted-nearby context, and missing or unused source groups.

Missing task flow:
1. User runs `brain context assemble` with no active session and no `--task`.
2. Brain returns a clear error explaining that a task or active session is required.

## Data / Interfaces

Public command surface:
- `brain context assemble --task "<task>"`
- `brain context assemble`
- `brain context assemble --explain`
- `brain context assemble --limit <n>`

Resolution rules:
- `--task` wins when provided.
- Otherwise use the active session task.
- If neither exists, fail clearly.

Default limit:
- `--limit` defaults to `8` selected items in the packet.

Default human output sections:
- `## Task Context`
- `## Selected Context`
- `## Ambiguities` only when non-empty

Explain-mode-only human output sections:
- `## Why This Was Selected`
- `## Omitted Nearby Context`
- `## Missing Or Unused Source Groups`

JSON output contract:

```json
{
  "task": {
    "text": "tighten auth flow",
    "source": "flag"
  },
  "summary": {
    "confidence": "high",
    "selected_count": 0,
    "group_counts": {
      "durable_notes": 0,
      "generated_context": 0,
      "structural_repo": 0,
      "live_work": 0,
      "policy_workflow": 0
    }
  },
  "selected": {
    "durable_notes": [],
    "generated_context": [],
    "structural_repo": [],
    "live_work": [],
    "policy_workflow": []
  },
  "ambiguities": [],
  "omitted_nearby": {
    "durable_notes": [],
    "generated_context": [],
    "structural_repo": [],
    "live_work": [],
    "policy_workflow": []
  }
}
```

Per-item shape:

```json
{
  "source": "docs/project-overview.md",
  "label": "Project Overview",
  "kind": "note",
  "excerpt": "short snippet or summary",
  "why": "short human-readable reason"
}
```

Explain-mode item diagnostics:

```json
{
  "rank": 1,
  "selection_method": "search",
  "diagnostics": {
    "source_group": "durable_notes",
    "notes": ["matched task terms", "high search rank"]
  }
}
```

Source groups:
- `durable_notes`
- `generated_context`
- `structural_repo`
- `live_work`
- `policy_workflow`

First-wave population:
- Populate `durable_notes`, `generated_context`, and `policy_workflow` in this epic.
- Keep `structural_repo` and `live_work` in the schema but allow them to remain empty until later epics land.

Source-group membership in the first wave:
- `durable_notes`: `docs/**/*.md`, `.brain/**/*.md` excluding `.brain/context/*`, and `AGENTS.md` when selected as repo truth instead of workflow guidance.
- `generated_context`: `.brain/context/current-state.md`, `.brain/context/overview.md`, `.brain/context/architecture.md`, `.brain/context/standards.md`.
- `policy_workflow`: `AGENTS.md` required workflow content, `.brain/policy.yaml`, `.brain/context/workflows.md`, `.brain/context/memory-policy.md`.

## Assembly Pipeline

1. Resolve task text from `--task` or active session.
2. Load the compact static base from the current context primitives.
3. Run current markdown search using the resolved task and active-task bias when a session exists.
4. Classify candidate sources into packet source groups.
5. Select up to `8` total items with these default caps:
   - up to `3` durable-note items
   - up to `2` generated-context items
   - up to `2` policy/workflow items
   - remaining capacity can be filled by any non-empty group
6. Deduplicate by source path plus heading.
7. Build ambiguity notes when:
   - there is no active session and task resolution relies only on `--task`
   - only one source group contains useful context
   - multiple nearby sources compete for the same role
8. Compute packet confidence:
   - `high`: at least three non-empty groups and no ambiguity notes
   - `medium`: two non-empty groups or one ambiguity note
   - `low`: zero or one non-empty groups, or two or more ambiguity notes
9. Build omitted-nearby lists from next-best candidates in each group.
10. Hide omitted-nearby detail in default human output and show it in `--explain` output.

## Risks / Open Questions

- How far can `context assemble` stretch before Brain eventually needs a separate packet-specific command family?
- Is the default cap of `8` items compact enough in real agent workflows?
- Are the confidence buckets useful as phrased, or do they need different language once the first implementation is exercised?

## Rollout

1. Add the `context assemble` command and stable packet schema.
2. Build the first implementation on top of current context and search primitives.
3. Use this epic to prove the packet UX before structural and live-work epics deepen the source groups.
4. Re-evaluate whether `context assemble` should remain the long-term surface or evolve further once the later epics land.

## Story Breakdown

- [ ] Add the `brain context assemble` command and stable packet schema.
- [ ] Implement first-wave typed assembly from current durable notes, generated context, and policy/workflow sources.
- [ ] Add explain-mode rationale, omitted-nearby reporting, and confidence output.

## Resources

- [[.brain/brainstorms/context-engine-v2.md]]
- [[.brain/planning/specs/retrieval-and-index-lifecycle.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]
- [[.brain/resources/references/skills-and-context-engineering.md]]

## Notes

This epic defines the user-facing task-context contract for the v2 initiative. The structural and live-work epics should enrich this packet shape rather than invent parallel context products.
