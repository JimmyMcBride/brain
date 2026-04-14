# Architecture

<!-- brain:begin context-architecture -->
Use this file for the structural shape of the repository.

## Internal Packages

- `internal/app/`
- `internal/backup/`
- `internal/brainstorm/`
- `internal/buildinfo/`
- `internal/config/`
- `internal/contextassembly/`
- `internal/distill/`
- `internal/embeddings/`
- `internal/history/`
- `internal/index/`
- `internal/livecontext/`
- `internal/notes/`
- `internal/output/`
- `internal/plan/`
- `internal/project/`
- `internal/projectcontext/`
- `internal/search/`
- `internal/session/`
- `internal/skills/`
- `internal/structure/`
- `internal/templates/`
- `internal/update/`
- `internal/workspace/`

## Architecture Notes

- Favor small package boundaries and explicit CLI/app wiring.
- Keep public CLI behavior stable; add internal seams only when they improve testability or safety.
- Treat generated project context as deterministic repo state, not LLM-authored prose.
- Treat session enforcement as the hard-control layer above soft context files.
<!-- brain:end context-architecture -->

## Local Notes

Add repo-specific notes here. `brain context refresh` preserves content outside managed blocks.
