---
title: "Architecture And Code Map"
type: "reference"
created: "2026-04-11T00:00:00Z"
updated: "2026-04-11T00:00:00Z"
source: "migrated_project_memory"
---
# Architecture And Code Map

## High-Level Shape

Brain is a single Go CLI.

- `main.go` boots Cobra.
- `cmd/*` owns flag parsing and CLI orchestration.
- `internal/app` is the composition root.
- `internal/workspace` owns project-root validation and managed markdown discovery.
- `internal/notes` owns frontmatter, templates, note IO, and history-aware updates.
- `internal/index`, `internal/search`, and `internal/embeddings` own the local retrieval stack.
- `internal/projectcontext`, `internal/session`, `internal/plan`, and `internal/brainstorm` own the repo-local workflow layer.

## Important Boundaries

- Keep command handlers thin.
- Keep project-local markdown as the source of truth.
- Treat generated context as deterministic repo state.
- Prefer explicit seams that improve tests or safety, not abstraction for its own sake.
