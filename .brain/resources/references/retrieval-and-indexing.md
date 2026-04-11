---
title: "Retrieval And Indexing"
type: "reference"
created: "2026-04-11T00:00:00Z"
updated: "2026-04-11T00:00:00Z"
source: "migrated_project_memory"
---
# Retrieval And Indexing

## Retrieval Model

Brain indexes project-managed markdown only:

- `AGENTS.md`
- `docs/**/*.md`
- `.brain/**/*.md`

The index is local to the project and stored in `.brain/state/brain.sqlite3`.

## Core Packages

- `internal/index/chunk.go`
- `internal/index/sqlite.go`
- `internal/search/search.go`
- `internal/embeddings/provider.go`

## Operational Notes

- Search should stay explainable and local-first.
- FTS and embeddings are both part of the ranking path.
- Retrieval changes should be verified with `go test ./...`, `brain doctor`, and targeted `brain search` queries.
