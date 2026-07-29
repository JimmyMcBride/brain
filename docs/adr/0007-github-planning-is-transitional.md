# ADR 0007: GitHub Planning Support Is Transitional

- Status: Accepted
- Date: 2026-07-27

## Context

Plan uses GitHub Discussions, Issues, milestones, Projects, and reconciliation.
This enables collaboration now, but permanent canonical GitHub ownership would
constrain the Planning domain and compete with Brain Cloud.

## Decision

Retain GitHub behavior during migration, then place it behind optional
collaboration/publication/repository/external-work/execution adapters. Planning
domain types remain provider-neutral.

## Consequences

Existing GitHub workflows and metadata remain supported during transition.
Long-term GitHub focuses on repository evidence, PRs, and optional mirroring.

## Alternatives Considered

- Remove GitHub now: breaks collaboration and migration.
- Keep GitHub canonical forever: couples domain and blocks Cloud-neutral design.
- Copy behavior into Core: makes an integration mandatory.

## Migration Implications

Preserve IDs, links, action plans, confirmation, and idempotency while extracting
adapters in Phase 5.

## Follow-up Work

Create adapter contract tests and publish deprecation timing for source mode.
