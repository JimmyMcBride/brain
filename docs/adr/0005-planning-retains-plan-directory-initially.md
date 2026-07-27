# ADR 0005: Planning Retains `.plan/` Storage Initially

- Status: Accepted
- Date: 2026-07-27

## Context

Existing Plan workspaces store human-readable planning under `.plan/`.
Relocating it during product integration would cause churn, complicate dual-CLI
compatibility, and mix Planning data with Brain Core memory/runtime state.

## Decision

Initial local Brain Planning retains `.plan/` where practical. Planning owns its
schema through a scoped storage adapter. Brain does not automatically treat all
Planning artifacts as Core memory.

## Consequences

Existing repositories can migrate gradually and both CLIs can coexist. Two hidden
directories remain, with distinct ownership. Relocation may be reconsidered only
after the combined product stabilizes.

## Alternatives Considered

- Move under `.brain/`: cleaner branding, high churn and boundary confusion.
- Cloud-only migration: breaks local-first use.
- Duplicate/mirror both directories: conflicting truth.

## Migration Implications

Capture schema fixtures, detect versions, migrate idempotently, and preserve
unknown/old data with diagnostics.

## Follow-up Work

Implement local adapter and compatibility tests in Phase 3.
