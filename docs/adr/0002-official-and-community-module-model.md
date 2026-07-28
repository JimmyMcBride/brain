# ADR 0002: Brain Supports Official and Community Modules

- Status: Accepted
- Date: 2026-07-27

## Context

Teams need different integrations and workflows. Undocumented hooks or
Planning-specific APIs would couple extensions to private implementation.

## Decision

Brain defines common lifecycle, capability, configuration, permission, event,
context/search, memory-proposal, tool, job, API, and future web contracts.
Official and community modules use these contracts; official status grants no
private hook.

## Consequences

Core owns mediation and attribution. Domain behavior stays within its module.
Planning is the first major design test, not the definition of every extension.

## Alternatives Considered

- Hard-code official integrations: fast initially, closed ecosystem.
- Unversioned internal hooks: weak compatibility and security.
- Design only for Planning: produces falsely generic APIs.

## Migration Implications

Phase 1 implements only contracts required for a trivial module. Later Planning
needs add contracts incrementally.

## Follow-up Work

Maintain the requirements matrix and enforce dependency rules with tests.
