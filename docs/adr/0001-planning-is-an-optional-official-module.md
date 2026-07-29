# ADR 0001: Planning Is an Optional Official Module

- Status: Accepted
- Date: 2026-07-27

## Context

Brain is currently focused on project context, memory, retrieval, and workflow.
Standalone Plan has a mature planning domain, but merging it into Core would make
one delivery workflow mandatory and duplicate identity/context/cloud systems.

## Decision

Planning becomes an official module, disabled by default and first-class when
enabled. Brain Core remains complete without it. Standalone Plan remains the
reference and compatibility product during migration.

## Consequences

Planning uses stable Core contracts and has explicit lifecycle, permissions,
events, storage, context, and health. Core cannot depend on Planning.

## Alternatives Considered

- Keep Plan permanently separate: duplicates platform/cloud foundations.
- Put Planning in Core: forces product-management concepts on every user.
- Make Planning a privileged exception: prevents a credible module ecosystem.

## Migration Implications

Build module foundations first, retain `.plan/` and the Plan CLI, then extract
domain behavior in phases.

## Follow-up Work

Implement ADR 0003 framework and the roadmap in `docs/planning/implementation-roadmap.md`.
