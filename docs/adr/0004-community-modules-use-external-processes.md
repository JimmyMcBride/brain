# ADR 0004: Community Modules Use an External-Process Model

- Status: Accepted
- Date: 2026-07-27

## Context

Community modules require language independence, separate releases, crash
containment, mediated authority, health, and version compatibility. Loading
untrusted code into Brain's process cannot provide these properties.

## Decision

Future community modules run as external processes over a versioned protocol.
Transport is not selected. Connect RPC, gRPC, and JSON-RPC remain candidates.

## Consequences

Brain needs lifecycle supervision, compatibility negotiation, structured logs,
health, timeouts, safe shutdown, and access mediation. No sandbox claim is made
until enforced.

## Alternatives Considered

- Go `plugin`: platform/version and isolation limitations.
- WASM now: promising isolation but premature runtime/API constraint.
- In-process third-party packages: unacceptable trust and release coupling.

## Migration Implications

External loading waits until compiled official modules validate contracts.

## Follow-up Work

Phase 11 prototypes transport and measures streaming, schemas, cancellation, SDK
quality, startup, Windows support, and debugging.
