# Architecture

`brain` is a local-first Go CLI for managing an Obsidian-compatible markdown vault with PARA at the top level and hybrid retrieval on top.

## Components

- `internal/config`: XDG-aware config loading plus env overrides.
- `internal/vault`: vault validation, PARA scaffolding, markdown walking, and path resolution.
- `internal/notes`: note model, YAML frontmatter handling, templates, file operations.
- `internal/index`: SQLite schema, FTS5 virtual table, markdown chunking, embedding storage.
- `internal/search`: hybrid search that merges FTS and embedding similarity.
- `internal/history` and `internal/backup`: append-only logs, pre-change backups, undo.
- `internal/content`: simple seed, gather, outline, and publish workflow.
- `internal/projectcontext`: generates repo-local `AGENTS.md`, `.brain/context/*`, and agent wrapper files.
- `internal/skills`: installs canonical and wrapper skill docs into agent directories.

## Project Context

`brain` also supports repo-local context engineering for coding agents:

1. `brain context install` creates a root `AGENTS.md`.
2. It generates a modular `.brain/context` bundle for overview, architecture, standards, workflows, memory policy, and current state.
3. It can generate thin agent-specific wrappers such as `.codex/AGENTS.md` or `.claude/CLAUDE.md`.
4. `brain context refresh` updates brain-managed sections while preserving user-authored content outside managed blocks.

## Hybrid RAG

The retrieval layer is intentionally simple and local:

1. Notes are parsed from the vault.
2. Markdown bodies are chunked on headings.
3. Chunks are written into SQLite and mirrored into an FTS5 table.
4. Embeddings are generated per chunk and stored as blobs.
5. Searches query FTS and embeddings separately.
6. Scores are normalized and merged into a final ranking.

The default embedding provider is `localhash`, which keeps the tool usable without network access. `openai` is supported for stronger semantic retrieval when `OPENAI_API_KEY` is available.

## PARA Model

`brain` keeps the top level intentionally narrow:

- `Projects/`: active outcomes with an end state.
- `Areas/`: ongoing responsibilities and recurring context.
- `Resources/`: reference material, captures, lessons, and content packages.
- `Archives/`: inactive material retained for history.

Richer structure belongs below those folders, not beside them.
