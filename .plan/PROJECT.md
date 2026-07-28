# Project: brain

Created: 2026-07-27T21:38:28Z

## Vision

Brain is an extensible context, memory, retrieval, and workflow platform for AI
agents. Official and community modules adapt Brain to different domains without
forcing one toolchain on every project. Planning becomes the first official
optional module while the standalone Plan product remains compatible during a
controlled migration.

## Principles

- Brain Core remains fully useful without Planning.
- Optional modules feel native when enabled and stay absent when disabled.
- Stable, permissioned contracts replace private cross-package hooks.
- Local-first remains valid; cloud and hybrid are additive.
- Human-readable durable artifacts remain important.
- Migration preserves behavior before changing ownership.

## Constraints

- No major Planning code migration before the internal module framework is
  reviewed.
- Initial local Planning retains `.plan/`.
- The standalone `plan` CLI and repository remain during migration.
- Supported migration source modes are local, GitHub, and GitHub hybrid. Brain
  Planning never imports or implements Linear integration.
- Planning cannot silently write Brain memory.
- Community modules are future external processes; no sandbox or protocol
  transport is claimed yet.

## Planning Rules

- Specs are the canonical execution contract.
- Promotion and integration mutations are preview-first, explicit, and
  idempotent.
- Runtime slices stay execution-ready and verification-aware without requiring
  permanent tiny files.
- Brain source references retain provenance and freshness.
- Planning approval and Brain memory approval remain separate.

## Notes

- Phase 0 architecture is documented in `docs/modules/`, `docs/planning/`, and
  ADRs 0001–0011.
- Planning is not implemented in Brain yet.
- The exact next step is the minimal compiled internal module framework and one
  trivial reference module.
