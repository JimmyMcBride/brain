# ADR 0008: Official Linear Integration Is Retired

- Status: Accepted
- Date: 2026-07-27

## Context

Standalone Plan contains a Linear source mode, metadata, and agent/MCP-mediated
promotion direction. Maintaining official parity would preserve unused provider
concepts and broaden product commitments.

## Decision

Retire official Linear support. Do not carry Linear-specific concepts into
generic contracts. Existing metadata receives explicit detection, export/migration,
and unsupported-configuration diagnostics. A future community module may support
Linear.

## Consequences

Product scope shrinks and generic APIs remain evidence-driven. Old workspaces need
a careful compatibility path.

## Alternatives Considered

- Keep official support: ongoing auth/API/workflow commitment.
- Delete immediately: risks data loss and opaque failures.
- Genericize every Linear feature: false abstraction.

## Migration Implications

Architecture marks retirement now; active code removal waits for Phase 6
diagnostics and migration fixtures.

## Follow-up Work

Execute `docs/planning/linear-removal.md`.
