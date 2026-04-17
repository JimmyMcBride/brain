---
args:
    - description: The project-memory or Brain workflow task to perform.
      name: task
      required: false
description: Use this skill when working with a project-local Brain workspace managed by the `brain` CLI, especially for repo memory, retrieval, compiled task context, session hygiene, and safe markdown updates.
name: brain
updated: "2026-04-16T22:00:00Z"
user-invocable: true
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
- support compiled context, durable memory, and session workflows inside the repo

## First Checks

When starting work in a repo that uses Brain:

1. Run `brain doctor`.
2. Read `project_migrations` in `brain doctor` when the repo may be older or was just upgraded.
3. Read `index_freshness` in `brain doctor` when retrieval matters.
4. Run `brain find <project>` or `brain search "<project> <task>"` when project memory matters.
5. Use `brain search status` before retrieval debugging so you know whether the local index is `fresh`, `stale`, or `missing`.
6. Read the repo contract and relevant docs before substantial work.

## Command Guide

Use these commands by default:

- `brain init --project .`
  - Create the local Brain workspace for a project.
- `brain doctor --project .`
  - Validate the local workspace, sqlite state, project migration state, and embedder configuration.
- `brain update --project .`
  - Update the Brain binary, refresh already-installed Brain skills, and apply pending project migrations for the current Brain repo.
- `brain context migrate --project .`
  - Run project migrations explicitly with the current binary when upgrade hygiene matters.
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
- `brain context compile --task "..."`
  - Compile the smallest summary-first working-set packet Brain can justify for the task, including anchors, provenance, nearby tests, verification hints, and budget diagnostics.
- `brain context compile --task "..." --budget small`
  - Ask for a leaner startup packet with a named preset or an explicit integer token target.
- `brain context compile --task "..." --fresh`
  - Bypass session-local packet reuse and force a standalone full packet when debugging or inspecting packet contents.
- `brain context explain --last`
  - Inspect the latest recorded compiled packet, including cache status, reuse or delta lineage, later expansions, and downstream session outcomes.
- `brain context stats`
  - Summarize likely signal, likely noise, repeated expansions, verification links, fresh-packet budget pressure, and recurring omitted markdown docs from local compiler telemetry.
- `brain distill --session`
  - Create a session-scoped promotion-review proposal with source provenance, promotion diagnostics, and suggested durable note updates.
- `brain context structure`
  - Inspect the derived structural repo map of boundaries, entrypoints, config surfaces, and test surfaces.
- `brain context structure --path "..."`
  - Focus the structural repo map on one subtree such as `internal/search`.
- `brain context structure status`
  - Inspect structural cache freshness and counts without rebuilding it.
- `brain context live --task "..."`
  - Inspect the current boundary-aware live-work packet for a task using the active session when available.
- `brain context live --explain`
  - Add rationale and missing-signal reporting for the live-work packet.
- `brain context assemble --task "..."`
  - Assemble a task-focused context packet from durable notes, generated context, structural repo context, and workflow/policy sources.
- `brain context assemble --explain`
  - Add rationale, omitted-nearby context, missing-group reporting, ambiguities, and confidence to the task packet.
- `brain context load --level 0`
  - Load the compatibility static bundle when an older level-based workflow is still required.
- `brain context load --level 1`
  - Add overview and workflows to the compatibility static bundle.
- `brain context load --level 2`
  - Load the full compatibility static context bundle.
- `brain context load --level 3 --query "..."`
  - Add search-injected relevant context to the compatibility bundle. If a session is active, the task can stand in for `--query`.
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

## Upgrade Workflow

- `brain update` refreshes already-installed global Brain skills, already-installed local Brain skills inside the current `--project`, and pending project migrations for the current Brain repo.
- Other Brain repos repair local Brain skills and apply only auto-safe project migrations lazily the next time app-backed Brain commands run there.
- Brain treats `.brain/session.json`, `.brain/sessions/`, `.brain/state/`, and `.brain/policy.override.yaml` as local runtime state. The durable shared layer is the markdown/docs surface, not the raw runtime trace.
- Explicit upgrade actions such as `brain update` and `brain context migrate` may refresh `.gitignore` and remove legacy tracked runtime-state files from the Git index while keeping them on disk. Review and commit that diff after the command reports it.
- If automatic project migration fails, run `brain doctor --project .`; then `brain context refresh --project .`; run `brain adopt --project .` if existing local agent files still need their Brain-managed integration block refreshed or migrated.

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

1. `brain context compile --task "<task>"` for the smallest justified startup packet.
2. Add `--budget small|default|large|<integer>` when you need a tighter or wider packet and want Brain to explain budget pressure explicitly.
3. Expect compile to reuse the latest matching packet inside the active session when relevant compile inputs are unchanged, and to emit a compact delta when the task is stable but the packet changed.
4. Add `--fresh` when you need to bypass that session-local reuse or delta behavior and force a standalone full packet.
5. `brain context explain --last` when you need to inspect why the latest packet looked the way it did, whether Brain reused or delta-linked it, which items were expanded later, or which downstream verification and closeout outcomes were recorded.
6. `brain context stats` when you are tuning compiler behavior and want a compact view of likely signal, likely noise, repeated expansions, verification-link patterns, fresh-packet budget pressure, and recurring omitted docs from local telemetry.
7. `brain context structure` when you need repo boundaries, entrypoints, config surfaces, or test surfaces before deeper retrieval.
8. `brain context live --task "<task>"` when you need current session, changed-file, touched-boundary, nearby-test, verification-recipe, or policy signals, not just compiled startup context.
9. `brain context assemble --task "<task>"` when you need the broader typed packet instead of the compiler-first working set.
10. `brain context assemble --explain` when you need to inspect why Brain chose its broader packet and what it left nearby.
11. `brain context load --level ...` only when you need the older static-bundle compatibility view.

## Distillation Workflow

1. Run `brain distill --session` when a working session surfaced decisions, tradeoffs, bugs, or discoveries that should become durable memory.
2. Review the proposal note under `.brain/resources/changes/`, including the promotion review section and the suggested targets that were actually classified as promotable.
3. Apply the durable note updates with `brain edit` or by updating the target notes directly after review.
4. Treat distill as a proposal generator, not as an auto-write path.

## Session Recovery

- If `brain session finish` blocks, inspect the promotion suggestions in the closeout output first.
- Run `brain distill --session` when you need the full promotion review note before deciding what to keep.
- Review the proposal, update the durable notes that matter, then retry `brain session finish`.
- If the session changed no durable knowledge after review, use `brain session finish --force --reason "<why>"` explicitly instead of pretending there was a durable update.
- Keep using `brain session run -- <command>` for required verification commands before closeout.

## When Not To Use This Skill

- pure software engineering tasks unrelated to project memory
- cases where the user explicitly wants raw filesystem work instead of Brain-managed notes
- repos that do not use Brain and do not need project-local memory workflow
