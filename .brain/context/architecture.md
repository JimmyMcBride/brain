---
updated: "2026-09-07T01:13:52Z"
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

- 2026-09-07: Phase 5 collaboration services assess external source content and contributions, preview canonical Specs repairs, and require read/collaboration grants, a preview revision, confirmation, and an audit sink before applying. Events carry an optional external source reference rather than inventing a local artifact identity. The GitHub adapter owns pagination, source identity checks, redacted errors, and a fresh pre-write revision check; GitHub has no atomic conditional Discussion update, so the final read/write race remains explicit. Publication/adoption and native host wiring remain pending.

- 2026-09-06: `planning/github` is the optional Phase 5 provider package. It exposes capability views for the provider-neutral application ports, owns an injected context-aware `gh` runner, and reads/writes exact current `.plan/.meta/github.json` structure atomically without credential fields. Construction and every disabled operation perform zero process, filesystem, network, or metadata work; enabled capabilities remain explicitly unsupported until migrated behind conformance tests.
- 2026-09-06: `planning/application/integration.go` owns the Phase 5 provider-neutral external reference, collaboration, publication, repository evidence, durable mapping, adoption/reconciliation, execution workspace, and typed integration-error contracts. The five capability-sized ports depend only on Planning domain and stdlib; application code owns policy and action classification, while provider adapters return opaque evidence. Exact export, dependency, and provider-noun guards protect this boundary before the provider adapter package exists.
- 2026-09-06: `internal/planning/conformance/testdata/github-v1` is a separate Phase 5 baseline pinned to standalone Plan `v0.1.29`. Its manifest, provider fixtures, transcripts, output projections, remote identities, metadata snapshots, failure recovery, and rerun contracts constrain provider-neutral port extraction without changing the older Phase 4 local baseline. Required tests are fully fake-backed and make no live GitHub calls.
- 2026-05-16: `internal/projectcontext/manager.go` owns the base `AGENTS.md` template emitted by `brain adopt` and `brain context refresh`; keep generated contract behavior, Karpathy guidelines, and post-adoption enrichment guidance there with matching projectcontext goldens.
- 2026-05-16: `internal/projectcontext/guidance.go` stores local optional guidance decisions in Brain state; `cmd/update.go` reports unset Karpathy Guidelines decisions to the AI agent, and `cmd/context.go` records accept/decline/status decisions.
- 2026-07-29: `internal/modules` now attributes command groups and typed events to compiled descriptors, rejects collisions, and gates command resolution through project enablement and exact grants. `main.go` registers `official.planning` at the outer composition root; disabled modules remain uninstantiated.
- 2026-07-31: Public `planning` owns the stdlib-only storage-neutral domain. Public `planning/application` owns shared use cases, DTOs, repository and host-policy contracts; public `planning/local` owns schema-v3 filesystem persistence and depends only on domain, application, stdlib, and YAML. External-package tests lock all three export/dependency boundaries and reject `go.mod` replacements.
- 2026-07-29: `internal/planning/conformance` owns the Phase 4 compatibility manifest, the pinned standalone Plan revision, command dispositions, normalization rules, copied fixtures, and black-box goldens. It remains internal while command families move behind captured contracts.
- 2026-07-29: Brain's canonical Go module path is `github.com/JimmyMcBride/brain`. The Phase 4 normalization changes imports plus linker targets in the release workflow and maintainer refresh scripts only.
- 2026-08-09: `cmd/plan.go` is a bounded native shell over the registered public Planning application service. In addition to status/check/roadmap and basic artifact reads, it exposes local brainstorm capture/refinement/challenge, guided-session navigation and guide packets, roadmap parking, maturity/source repair, and direct brainstorm-to-spec promotion. Shared mutations require explicit confirmation, module permission, durable audit, atomic local writes, and idempotent reruns. Direct local promotion creates canonical specs without epic or persisted story intermediates; commands still fail with `module_disabled` until `official.planning` is granted and enabled.
- 2026-08-09: Canonical local spec edit, approval, analysis, checklist, initiative, execution, and guided handoff now run through public `planning/application` plus `planning/local`. Execution slices are deterministic and ephemeral; handoff compensates the exact spec transition if guided-session persistence fails.
- 2026-07-27: Planning is an official optional module. It retains `.plan/` for initial local compatibility and supports `local`, `github`, and `hybrid` migration sources; during standalone Plan migration, `hybrid` means split local/GitHub ownership. Brain Planning never imports or implements Linear integration; legacy Linear workspaces receive standalone Plan migration guidance. Planning uses review-first Brain memory proposals.
