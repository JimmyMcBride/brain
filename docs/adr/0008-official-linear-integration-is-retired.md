# ADR 0008: Brain Planning Excludes Linear Integration

- Status: Accepted
- Date: 2026-07-27

## Context

Standalone Plan contains a Linear source mode, metadata, and agent/MCP-mediated
promotion direction. Maintaining official parity would preserve unused provider
concepts and broaden product commitments.

## Decision

Brain Planning supports `local`, `github`, and `hybrid` migration sources only.
It never imports or implements Linear integration. Do not carry Linear-specific
schemas, configuration, commands, permissions, events, adapters, or concepts into
generic contracts. A Linear-configured workspace receives an unsupported-source
diagnostic directing the user to migrate it with standalone Plan first.

## Consequences

Product scope shrinks and generic APIs remain evidence-driven. Brain avoids
becoming a second Linear migration runtime.

## Alternatives Considered

- Keep official support: ongoing auth/API/workflow commitment.
- Import read-only Linear metadata: still brings Linear schema into Brain.
- Genericize every Linear feature: false abstraction.

## Migration Implications

Standalone Plan owns safe cleanup/export. Brain Planning only detects the
unsupported source value at its workspace boundary and refuses enablement.

## Follow-up Work

Execute `docs/planning/linear-removal.md`.
