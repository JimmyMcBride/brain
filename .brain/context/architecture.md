---
updated: "2026-07-28T05:43:17Z"
---
# Architecture

<!-- brain:begin context-architecture -->
Use this file for the structural shape of the repository.

## Internal Packages

- `internal/app/`
- `internal/backup/`
- `internal/buildinfo/`
- `internal/config/`
- `internal/contextassembly/`
- `internal/contextaudit/`
- `internal/distill/`
- `internal/embeddings/`
- `internal/history/`
- `internal/index/`
- `internal/livecontext/`
- `internal/modules/`
- `internal/notes/`
- `internal/output/`
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
<!-- brain:end context-architecture -->

## Local Notes

- 2026-05-16: `internal/projectcontext/manager.go` owns the base `AGENTS.md` template emitted by `brain adopt` and `brain context refresh`; keep generated contract behavior, Karpathy guidelines, and post-adoption enrichment guidance there with matching projectcontext goldens.
- 2026-05-16: `internal/projectcontext/guidance.go` stores local optional guidance decisions in Brain state; `cmd/update.go` reports unset Karpathy Guidelines decisions to the AI agent, and `cmd/context.go` records accept/decline/status decisions.
- 2026-07-28: Phase 1 implements `internal/modules` as a flat compiled-module registry and runtime with tracked project config, ignored local grants, explicit lifecycle and health, `brain modules` administration, and a test-only module. Production registers zero modules; dependencies, dynamic commands, providers, events, external processes, and Planning remain deferred.
- 2026-07-27: Planning is an official optional module. It retains `.plan/` for initial local compatibility and supports `local`, `github`, and `hybrid` migration sources; during standalone Plan migration, `hybrid` means split local/GitHub ownership. Brain Planning never imports or implements Linear integration; legacy Linear workspaces receive standalone Plan migration guidance. Planning uses review-first Brain memory proposals.
