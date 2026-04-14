---
name: brain
description: Use this skill when working with a project-local Brain workspace managed by the `brain` CLI, especially for repo memory, retrieval, planning, brainstorming, agent context, and safe markdown updates.
user-invocable: true
args:
  - name: task
    description: The project-memory or Brain workflow task to perform.
    required: false
---

# Brain

Use `brain` as the primary interface for project-local memory and workflow.

## Project-First Behavior

If the current repository has Brain context, use the repo-local Brain docs first:

1. Read `AGENTS.md` at the repo root.
2. Read `.brain/policy.yaml`.
3. Read the linked `.brain/context/*.md` files needed for the task.
4. If no validated session is active, run `brain session start --task "<task>"`.
5. If a session is already active, run `brain session validate`.
6. Use the `brain` CLI for durable project-memory operations.
7. Fall back to this generic skill only when repo-local context is absent.

## Goals

- keep durable project knowledge in markdown
- prefer explicit CLI operations over ad hoc memory files
- preserve backups, history, and undo for note changes
- keep retrieval focused on repo-managed docs instead of transient files
- support planning, brainstorming, context, and session workflows inside the repo

## First Checks

When starting work in a repo that uses Brain:

1. Run `brain doctor`.
2. Read `index_freshness` in `brain doctor` when retrieval matters.
3. Run `brain find <project>` or `brain search "<project> <task>"` when project memory matters.
4. Use `brain search status` before retrieval debugging so you know whether the local index is `fresh`, `stale`, or `missing`.
5. Read the repo contract and relevant docs before substantial work.

## Command Guide

Use these commands by default:

- `brain init --project .`
  - Create the local Brain workspace for a project.
- `brain doctor --project .`
  - Validate the local workspace, sqlite state, and embedder configuration.
- `brain read <path>`
  - Read a managed markdown note.
- `brain edit <path> ...`
  - Update title, metadata, or body while preserving history and backups.
- `brain find [query]`
  - Search path, title, type, or note content.
- `brain search "query"`
  - Run hybrid retrieval over the local project index. With the default `localhash` provider, this is best understood as lexical search plus lightweight semantic hinting.
- `brain search status`
  - Inspect index freshness, indexed counts, and the local sqlite path without mutating the index.
- `brain search --explain "query"`
  - Show lexical and semantic score contributions plus the retrieval source classification for each result.
- `brain search --inject "query"`
  - Return ranked results plus an agent-ready `## Relevant Context` block that can be reused directly.
- `brain distill --session`
  - Create a session-scoped distill proposal note with source provenance and suggested durable note updates.
- `brain distill --brainstorm <path>`
  - Create the same style of proposal note from a brainstorm source.
- `brain brainstorm ...`
  - Manage project-local brainstorming notes.
- `brain plan ...`
  - Manage project-local epics, specs, and stories with a spec-driven workflow.
- `brain context load --level 0`
  - Load the minimal AGENTS summary plus current state.
- `brain context load --level 1`
  - Add overview and workflows to the minimal context bundle.
- `brain context load --level 2`
  - Load the full static context bundle.
- `brain context load --level 3 --query "..."`
  - Load the full static bundle plus search-injected relevant context. If a session is active, the task can stand in for `--query`.
- `brain context structure`
  - Inspect the derived structural repo map of boundaries, entrypoints, config surfaces, and test surfaces.
- `brain context structure --path "..."`
  - Focus the structural repo map on one subtree such as `internal/search`.
- `brain context structure status`
  - Inspect structural cache freshness and counts without rebuilding it.
- `brain context live --task "..."`
  - Inspect the current live-work packet for a task using the active session when available.
- `brain context live --explain`
  - Add rationale and missing-signal reporting for the live-work packet.
- `brain context assemble --task "..."`
  - Assemble a task-focused context packet from durable notes, generated context, structural repo context, and workflow/policy sources.
- `brain context assemble --explain`
  - Add rationale, omitted-nearby context, missing-group reporting, ambiguities, and confidence to the task packet.
- `brain context install --project .`
  - Create or adopt the root contract plus `.brain/context`.
- `brain context refresh --project .`
  - Refresh generated project context while preserving local notes outside managed blocks.
- `brain session start --project . --task "..."`
  - Start a validated project session.
- `brain session run --project . -- <command>`
  - Execute and record required verification commands.
- `brain session finish --project .`
  - Enforce policy and close the active session.
- `brain history`
  - Inspect tracked note operations.
- `brain undo`
  - Revert the latest tracked note operation.

## Operating Rules

- Prefer `brain edit` over direct mutation when the target is Brain-managed markdown.
- Keep durable project discoveries in `AGENTS.md`, `docs/`, or `.brain/`.
- Do not create sidecar memory systems when Brain already owns the workflow.
- Prefer updating an existing durable note over creating duplicates.
- Use human-readable filenames and titles.

## Retrieval Workflow

1. `brain find <keyword>` for quick narrowing.
2. `brain search "<task or concept>"` for ranked results.
3. `brain search status` when results look stale, missing, or surprising.
4. `brain search --explain "<task or concept>"` when you need to inspect ranking behavior.
5. `brain search --inject "<task or concept>"` when you need a compact context block to pass straight into the next step.
6. `brain read <path>` for the winning notes when the injected block is not enough.
7. Re-run search after meaningful note updates when you need the latest local state reflected. Brain will rebuild the local index automatically only when it is stale or missing.
8. If retrieval quality matters, check which provider is active in `brain doctor` or `brain search status` before assuming the project is using a strong hosted semantic model.

## Context Workflow

1. `brain context load --level 0` for minimal startup context.
2. `brain context load --level 1` when you need summaries and workflows.
3. `brain context load --level 2` when the task needs the full static context bundle.
4. `brain context load --level 3 --query "<task or concept>"` when you need search-driven deep context.
5. `brain context structure` when you need repo boundaries, entrypoints, config surfaces, or test surfaces before deeper retrieval.
6. `brain context live --task "<task>"` when you need current session, changed-file, nearby-test, verification, or policy signals, not just static repo context.
7. `brain context assemble --task "<task>"` when you need a task-focused packet instead of a raw static bundle.
8. `brain context assemble --explain` when you need to inspect why Brain chose its packet and what it left nearby.
9. Prefer requesting the next level explicitly instead of loading everything up front.

## Distillation Workflow

1. Run `brain distill --session` when a working session surfaced decisions, tradeoffs, bugs, or discoveries that should become durable memory.
2. Run `brain distill --brainstorm <path>` when a brainstorm should collapse into proposed durable note updates.
3. Review the proposal note under `.brain/resources/changes/`.
4. Apply the durable note updates with `brain edit` or by updating the target notes directly after review.
5. Treat distill as a proposal generator, not as an auto-write path.

## Session Recovery

- If `brain session finish` blocks on missing durable memory updates, run `brain distill --session`.
- Review the proposal, update the durable notes that matter, then retry `brain session finish`.
- Keep using `brain session run -- <command>` for required verification commands before closeout.

## When Not To Use This Skill

- pure software engineering tasks unrelated to project memory
- cases where the user explicitly wants raw filesystem work instead of Brain-managed notes
- repos that do not use Brain and do not need project-local memory workflow
