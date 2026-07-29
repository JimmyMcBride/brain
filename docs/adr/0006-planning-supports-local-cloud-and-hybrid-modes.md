# ADR 0006: Planning Supports Local, Cloud, and Hybrid Modes

- Status: Accepted
- Date: 2026-07-27

## Context

Brain's direction spans local, Brain Cloud, and hybrid use. Planning must not
require hosted infrastructure, yet teams need collaboration and agents across
environments.

## Decision

Planning supports local repository storage, Cloud-native storage, and explicit
layer-by-layer hybrid ownership. Planning sync is separate from Brain
context-memory sync.

## Consequences

Cloud mode needs no `.plan/`. Hybrid mode needs revisions, conflicts, offline
behavior, recovery, idempotency, and clear ownership. Capability absence remains
valid.

## Alternatives Considered

- Local only: blocks unified cloud product.
- Cloud only: violates local-first and migration needs.
- Whole-project mode switch: too coarse for practical hybrid workflows.

## Migration Implications

Local ships first. Cloud and hybrid follow only after domain/storage contracts
and Brain Cloud foundations exist.

## Follow-up Work

Define cloud revisions in Phase 8 and hybrid protocol in Phase 9.
