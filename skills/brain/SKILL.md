---
name: brain
description: Use this skill when working with a local knowledge vault managed by the `brain` CLI, especially for PARA-structured Obsidian markdown workflows, retrieval, capture, content packaging, and safe note edits. Do not use it for unrelated code tasks or for direct vault file edits when the `brain` command can perform the action.
user-invocable: true
args:
  - name: task
    description: The vault, project memory, retrieval, capture, or content workflow task to perform with brain.
    required: false
---

# Brain

Use `brain` as the primary interface for working with the knowledge vault.

## Project-first behavior

If the current repository has a project contract, read it before relying on the generic skill text:

1. Read `AGENTS.md` at the repo root if it exists.
2. Read `.brain/policy.yaml` if it exists.
3. Read the linked `.brain/context/*.md` files needed for the task.
4. If no validated session is active, run `brain session start --task "<task>"`.
5. If a session is already active, run `brain session validate`.
6. Use the `brain` CLI as the operational interface for project memory and vault work.
7. Fall back to this skill's generic guidance only when project-local context is absent.

## Goals

- Treat the Obsidian markdown vault as the source of truth.
- Preserve PARA at the top level only:
  - `Projects/`
  - `Areas/`
  - `Resources/`
  - `Archives/`
- Prefer `brain` commands over direct file edits so backups, history, and undo remain usable.
- Reindex deliberately when retrieval quality depends on newly added or edited notes.

## First checks

When starting work with a vault:

1. Run `brain doctor` to confirm config, vault, sqlite, and embedding setup.
2. If search appears stale or empty, run `brain reindex`.
3. Use `brain find` for path, metadata, or lightweight content lookup.
4. Use `brain search "query"` for hybrid lexical plus semantic retrieval.

When starting work in a code repository that uses `brain` project context:

1. Read the project `AGENTS.md`.
2. Read `.brain/policy.yaml`.
3. Read `.brain/context/overview.md`, `.brain/context/architecture.md`, `.brain/context/workflows.md`, and `.brain/context/memory-policy.md`.
4. Retrieve any relevant durable notes before making large changes.

## OpenClaw usage

OpenClaw should use this skill when the user asks it to remember project context, search an Obsidian/PARA vault, capture discoveries, maintain project memory, or prepare content from notes.

For OpenClaw installs, this skill should live at:

- Global: `~/.openclaw/skills/brain/SKILL.md`
- Project-local: `<project>/.openclaw/skills/brain/SKILL.md`

Validate discovery with:

```bash
openclaw skills list --json
```

## Command guide

Use these commands by default:

- `brain init`
  - Create config, vault scaffolding, sqlite DB, and data directories.
- `brain add "Title" --section ... --type ...`
  - Create a note from a template.
- `brain capture "Title" --body "..."`
  - Fast capture into `Resources/Captures/...`.
- `brain daily [YYYY-MM-DD]`
  - Create or open a daily note.
- `brain read <path>`
  - Read a note cleanly.
- `brain edit <path> ...`
  - Update title, metadata, or body.
- `brain move <path> <destination>`
  - Move a note while preserving history and backups.
- `brain find [query]`
  - Search by path, metadata, title, or note content directly from the vault.
- `brain search "query"`
  - Run hybrid retrieval over indexed note chunks.
- `brain content seed|gather|outline|publish`
  - Promote notes into a content workflow.
- `brain history`
  - Inspect tracked operations.
- `brain undo`
  - Revert the last tracked change.
- `brain context install --project .`
  - Create a repo-local `AGENTS.md` plus a modular `.brain/context` bundle.
- `brain context refresh --project .`
  - Refresh brain-managed project context files without overwriting user notes outside managed blocks.
- `brain session start --project . --task "..."`
  - Start a validated project session and write local session state.
- `brain session validate`
  - Confirm an active session exists and inspect finish-stage obligations.
- `brain session run -- <command>`
  - Execute and record verification commands for the active session.
- `brain session finish`
  - Enforce memory updates, reindex requirements, and required command runs before closing the session.

## Operating rules

- Do not create new top-level folders outside PARA unless explicitly asked.
- Prefer stable, readable note names.
- Prefer `brain add`, `brain edit`, `brain move`, and `brain capture` over manual file mutation.
- Use `brain organize` as a dry run first; do not apply archive-like moves unless the user clearly wants them.
- When notes are created or edited outside `brain`, run `brain reindex` before relying on `brain search`.

## Retrieval workflow

Use this sequence when gathering context for a task:

1. `brain find <keyword>` for quick narrowing.
2. `brain search "<task or concept>"` for ranked context.
3. `brain read <path>` for the winning notes.
4. If the user is shaping publishable material, use `brain content gather` or `brain content outline`.

## Output and safety

- Prefer human output unless the task clearly needs machine-readable results; then use `--json`.
- For multi-step note changes, keep `brain history` and `brain undo` in mind.
- If search returns no results and the vault should contain relevant material, tell the user to reindex or run it yourself.

## When not to use this skill

- Pure software engineering tasks unrelated to the knowledge vault.
- Cases where the user explicitly wants raw filesystem operations instead of `brain`.
- Situations where `brain` is unavailable or not configured and the task is blocked on broader setup outside the vault workflow.
