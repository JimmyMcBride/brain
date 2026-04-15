---
created: "2026-04-15T03:55:58Z"
epic: v4-context-compiler-ux-migration
project: brain
status: approved
title: V4 Context Compiler UX Migration Spec
type: spec
updated: "2026-04-15T05:31:27Z"
---
# V4 Context Compiler UX Migration Spec

Created: 2026-04-15T03:40:00Z

## Why

The architecture shift is not complete until the user-facing model changes too. Brain should stop teaching people to think in terms of static levels or broad document loading and instead teach them to compile the smallest useful packet for the current task.

## Problem

Older context-loading surfaces, docs, and skill guidance still reflect a more document-centric or level-driven model. If those remain dominant after the compiler architecture lands, the product will feel split between two competing stories.

## Goals

- Make `context compile` the primary context surface.
- Demote or wrap older static-level mental models behind compatibility layers where needed.
- Update docs, skill guidance, and project contract language to reflect the compiler architecture.
- Keep the migration practical and low-confusion for existing users.

## Non-Goals

- Breaking existing workflows abruptly.
- Deleting every older context command immediately.
- Repositioning Brain back toward planning or broad workflow ownership.
- Expanding `AGENTS.md` into a larger knowledge dump.

## Requirements

- Update docs and skills to teach the compiler model.
- Reframe older `context` surfaces as compatibility paths or secondary views where appropriate.
- Make the default guidance emphasize:
  - tiny base contract
  - compiled working set packets
  - inspectable provenance
  - explicit promotion into durable memory
- Keep compatibility behavior where needed during the transition.
- Make the compiler-era UX consistent across CLI help, docs, and Brain skill guidance.

## UX / Flows

New-user flow:
1. User asks how to get context for a task.
2. Brain and its docs point to `brain context compile` as the primary answer.
3. Users learn the compiler model first instead of static context levels.

Compatibility flow:
1. Existing users continue using older `context` commands.
2. Brain keeps those working where practical.
3. Help and docs gradually point them toward `context compile` as the preferred surface.

## Data / Interfaces

Migration surfaces likely affected:
- CLI help under `brain context`
- `docs/usage.md`
- `docs/architecture.md`
- `skills/brain/SKILL.md`
- `.brain/context/*` generated guidance where relevant
- `AGENTS.md` project contract wording where relevant

## Risks / Open Questions

- When is the compiler mature enough to become the primary recommendation instead of an advanced surface?
- Which legacy commands should remain first-class versus becoming wrappers or niche inspection tools?
- How aggressively should Brain retire the phrase “context levels” if compatibility still exists?

## Rollout

1. Update compiler-era docs and skill guidance.
2. Reframe CLI help and preferred flows around `context compile`.
3. Keep compatibility paths where needed while demoting the old mental model.
4. Audit generated project context language so it matches the new architecture.

## Story Breakdown

- [ ] Update Docs And The Brain Skill To Teach The Compiler Model
- [ ] Reframe CLI Help And Compatibility Guidance Around Context Compile
- [ ] Audit Generated Guidance And Reduce Static-Level Language

## Resources

- [[.brain/brainstorms/context-substrate-direction.md]]
- [[.brain/brainstorms/context-compiler-rollout.md]]
- [[.brain/planning/specs/task-context-assembly.md]]

## Notes

This epic is where the product stops feeling like a document loader even to users who never read the internals.
