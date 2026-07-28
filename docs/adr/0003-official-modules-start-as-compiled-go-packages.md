# ADR 0003: Official Modules Start as Compiled Go Packages

- Status: Accepted
- Date: 2026-07-27

## Context

Brain is a single Go binary with centralized Cobra/app wiring. A new process
protocol would add distribution, supervision, and compatibility work before
extension contracts are proven.

## Decision

Stage 1 official modules are compiled Go packages registered through stable,
controlled interfaces. They are explicitly enabled, declare capabilities and
permissions, isolate domain tests, and distinguish local/cloud support.

## Consequences

Deployment stays simple. In-process modules share OS authority, so interfaces,
import rules, permission tests, and audit are the initial boundary—not a claimed
sandbox.

## Alternatives Considered

- External processes immediately: premature protocol and operations work.
- Go native `plugin`: portability, versioning, and isolation are unsuitable.
- Direct imports and command wiring: no enforceable module contract.

## Migration Implications

Add `internal/modules/*` and one trivial reference module before Planning code.
Avoid wholesale Core reorganization.

## Follow-up Work

Prototype registry, enablement, configuration, permissions, lifecycle, and health.
