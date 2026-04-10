---
name: brain
description: Use this skill when working with a local knowledge vault managed by the `brain` CLI, especially for PARA-structured Obsidian markdown workflows, retrieval, capture, content packaging, and safe note edits.
user-invocable: true
args:
  - name: task
    description: The vault, project memory, retrieval, capture, or content workflow task to perform with brain.
    required: false
---

# Brain Skill

Use this skill when the working source of truth is an Obsidian-style markdown vault managed by `brain`.

## Operating model

- Treat the vault as authoritative. Read and write markdown notes instead of inventing parallel state.
- Keep top-level organization in PARA only: `Projects/`, `Areas/`, `Resources/`, `Archives/`.
- Prefer explicit note links like `[[Projects/foo.md]]` when packaging related context.
- Reindex after major note changes when retrieval quality matters.
- Use `brain search` for semantic-plus-keyword retrieval and `brain find` for path or metadata lookups.

## High-value commands

- `brain init`: create config, PARA directories, sqlite index, and data dirs.
- `brain add`: create structured notes from templates.
- `brain capture`: add fast inbox-style notes under `Resources/Captures/...`.
- `brain daily`: create or open a dated daily note.
- `brain reindex`: rebuild FTS and embedding data from the vault.
- `brain search "query"`: hybrid retrieval over chunks.
- `brain content seed|gather|outline|publish`: move notes into a content workflow.
- `brain history` and `brain undo`: inspect and revert tracked operations.

## Suggested agent workflow

1. Read config or run `brain doctor`.
2. Use `brain find` or `brain search` to gather context.
3. Create or update notes with `brain add`, `brain edit`, `brain capture`, or `brain daily`.
4. Run `brain reindex` when new material should become searchable.
5. Use `brain content outline` when turning knowledge into publishable material.

## Guardrails

- Prefer edits through `brain` so backups and history are recorded.
- Treat `Archives/` as explicit, not automatic, unless the user asks for archiving.
- Keep filenames stable and human-readable.
- Avoid creating new top-level folders outside PARA.
