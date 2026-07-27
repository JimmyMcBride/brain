# ADR 0009: The Plan CLI Gets a Compatibility Migration Path

- Status: Accepted
- Date: 2026-07-27

## Context

Users and agents may depend on `plan` commands, flags, exit codes, JSON, and
`.plan/` artifacts. Immediate replacement would fork behavior or break scripts.

## Decision

Introduce `brain plan` in stages. Both command families call shared Planning
services. `plan` becomes a compatibility wrapper before separate distribution is
deprecated.

## Consequences

Migration takes longer but avoids behavioral forks. Warning output and machine
contracts need explicit tests and version policy.

## Alternatives Considered

- Flag day rename: unacceptable breakage.
- Maintain two implementations: inevitable drift.
- Keep only `plan`: misses native Brain experience.

## Migration Implications

Inventory and golden-test commands/JSON, map namespaces, then add warnings without
corrupting stdout.

## Follow-up Work

Implement Phases 3–4 and document support/removal timing.
