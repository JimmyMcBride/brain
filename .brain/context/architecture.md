---
updated: "2026-07-31T19:32:16Z"
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
- `internal/official/`
- `internal/output/`
- `internal/planning/`
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
- 2026-07-29: `internal/modules` now attributes command groups and typed events to compiled descriptors, rejects collisions, and gates command resolution through project enablement and exact grants. `main.go` registers `official.planning` at the outer composition root; disabled modules remain uninstantiated.
- 2026-07-31: Public `planning` owns the complete storage-neutral domain and imports only the standard library. Its canonical-path external test locks the reviewed export surface and rejects `go.mod` replacement directives. `internal/planning/application` still owns local use cases and mutation contracts; `internal/official/planning/local` still owns schema-v3 persistence and compatibility diagnostics.
- 2026-07-29: `internal/planning/conformance` owns the Phase 4 compatibility manifest, the pinned standalone Plan revision, command dispositions, normalization rules, copied fixtures, and black-box goldens. It remains internal while command families move behind captured contracts.
- 2026-07-29: Brain's canonical Go module path is `github.com/JimmyMcBride/brain`. The Phase 4 normalization changes imports plus linker targets in the release workflow and maintainer refresh scripts only.
- 2026-07-29: `cmd/plan.go` is a bounded native shell over the registered Planning application service. It exposes status, brainstorm list/show/start, and spec list/show; direct commands fail with `module_disabled` until `official.planning` is granted and enabled.
- 2026-07-27: Planning is an official optional module. It retains `.plan/` for initial local compatibility and supports `local`, `github`, and `hybrid` migration sources; during standalone Plan migration, `hybrid` means split local/GitHub ownership. Brain Planning never imports or implements Linear integration; legacy Linear workspaces receive standalone Plan migration guidance. Planning uses review-first Brain memory proposals.
