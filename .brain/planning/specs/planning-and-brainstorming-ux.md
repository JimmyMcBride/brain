---
created: "2026-04-11T21:53:09Z"
epic: planning-and-brainstorming-ux
project: brain
status: approved
title: Planning And Brainstorming UX Spec
type: spec
updated: "2026-04-13T06:23:52Z"
---
# Planning And Brainstorming UX Spec

Created: 2026-04-11T21:53:09Z

## Why

Brain is strongest when durable project memory stays explicit, local, and reviewable. The main weakness in that model is that valuable reasoning often never becomes durable memory at all. This planning pass focuses on closing that gap without sliding into transcript hoarding.

## Problem

Today Brain depends on explicit `brain edit` discipline and manual note updates. That preserves correctness, but it also means decisions, tradeoffs, bugs, and discoveries from active sessions can disappear unless someone remembers to capture them. The planning and brainstorming flow does not yet provide a first-class bridge from session work to curated project memory.

## Goals

- Add an optional memory distillation workflow that turns session activity into proposed durable-note updates.
- Preserve decision rationale as a first-class artifact instead of only capturing final outcomes.
- Keep Brain's explicit approval model intact so distillation never becomes silent background storage.
- Make the new memory-capture path feel native to the existing brainstorm -> epic -> spec -> story workflow.

## Non-Goals

- Storing full conversations or command transcripts as durable memory by default.
- Auto-writing AGENTS, context, or resource notes without review.
- Replacing existing explicit note-edit workflows for users who prefer manual capture.

## Requirements

- Add a `brain distill` workflow that can inspect recent session history, command runs, and git diff.
- Distillation must output candidate updates for Brain-managed targets such as `AGENTS.md`, `.brain/context/*`, `.brain/resources/changes/*`, and decision notes.
- Introduce a first-class decision note shape under `.brain/resources/decisions/` with enough structure to preserve context, options, decision, and tradeoffs.
- Keep approval in the loop so users or agents can accept, reject, or edit generated updates before they become durable memory.
- Record enough provenance that a later reader can tell which session or source material drove the distilled update.

## UX / Flows

Manual distillation flow:
1. User or agent runs `brain distill`.
2. Brain gathers recent session history, command runs, and git diff context.
3. Brain proposes note updates grouped by destination and reason.
4. User or agent reviews, edits, and accepts the proposed updates.

Decision-note flow:
1. User or distill flow creates a decision note.
2. The note captures context, options considered, final choice, and tradeoffs.
3. The note is linked from specs, stories, or change notes where relevant.

## Data / Interfaces

- Add a decision-note template and note-type conventions under `.brain/resources/decisions/`.
- Define a distillation payload shape that can represent proposed file targets, generated body changes, and source provenance.
- Keep generated output text-first and markdown-native so it fits the rest of Brain's note model.

## Risks / Open Questions

- How much source material should distill inspect by default before the output becomes noisy?
- Should approval happen inline in the terminal, through note creation, or via staged patch-like files?
- How deterministic does the distilled proposal need to be for tests and history expectations?

## Rollout

1. Add note templates and data structures for decision notes and distillation proposals.
2. Ship a manual `brain distill` path first.
3. Reuse the same primitives from session closeout once the manual workflow feels stable.

## Story Breakdown

- [ ] Add Session-Aware Memory Distillation
- [ ] Capture Decision Rationale As First-Class Notes

## Resources

- [[.brain/brainstorms/mempalace-inspired-brain-improvements.md]]
- [[.brain/planning/stories/planning-and-brainstorming-ux-current-state-and-next-actions.md]]

## Notes

The guiding constraint is simple: recover missing insight without turning Brain into a raw transcript archive.
