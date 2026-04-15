# Project Architecture

<!-- brain:begin project-doc-architecture -->
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
- `internal/promotion/`
- `internal/search/`
- `internal/session/`
- `internal/skills/`
- `internal/structure/`
- `internal/taskcontext/`
- `internal/templates/`
- `internal/update/`
- `internal/workspace/`

## Architecture Notes

- Favor small package boundaries and explicit CLI/app wiring.
- Keep public CLI behavior stable; add internal seams only when they improve testability or safety.
- Treat generated project context as deterministic repo state, not LLM-authored prose.
- Treat session enforcement as the hard-control layer above soft context files.
<!-- brain:end project-doc-architecture -->

## Local Notes

Important reference notes:

- [.brain/resources/references/architecture-and-code-map.md](../.brain/resources/references/architecture-and-code-map.md)
- [.brain/resources/references/retrieval-and-indexing.md](../.brain/resources/references/retrieval-and-indexing.md)
- [.brain/resources/references/skills-and-context-engineering.md](../.brain/resources/references/skills-and-context-engineering.md)

Current architecture emphasis:

- keep `internal/workspace` as the project-root boundary
- keep indexing scoped to Brain-managed markdown
- keep project context and session enforcement deterministic
