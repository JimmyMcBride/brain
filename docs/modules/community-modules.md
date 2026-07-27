# Community Modules

## Direction

Community modules are future external processes, not Go native plugins and not
arbitrary packages linked into the Brain binary. Official compiled modules must
first validate the extension contracts.

Community modules may add integrations, custom workflows, context/search
providers, memory types and proposals, agent tools, domain models, background
jobs, importers, publishers, approval or execution providers, and company policy.

## External Process Model

The future protocol must support:

- language-independent implementations and independent releases
- handshake with module ID/version and Brain API/protocol ranges
- capability and schema discovery
- explicit enablement, configuration, permission grants, and dependency checks
- request-scoped project/user identity
- lifecycle, health, logs, cancellation, timeouts, and safe shutdown
- permission-mediated filesystem, network, secret, and cloud access
- typed errors and compatibility diagnostics

Connect RPC, gRPC, and JSON-RPC remain transport candidates. Transport selection
must follow a prototype measuring streaming, schemas, cancellation, generated SDK
quality, local startup, Windows support, and debugging.

## Distribution

No registry or marketplace is approved. A future distribution design must cover
publisher identity, source, checksum, signature, supported platforms, version
pinning, revocation, allowlists, dependency resolution, update review, and
rollback.

## Planning Extensions

Planning may expose domain-specific extension points for readiness checks,
security/architecture/cost/compliance reviews, spec sections, estimates,
publishers, execution trackers, deployment gates, and organization approvals.
Those contracts belong to Planning. General permissions, events, tools, jobs,
configuration, and audit stay in Core.

The test for a Core extension point:

> Could an incident-management, architecture-analysis, or compliance-review
> module use it without pretending to be Planning?

If not, keep it in the Planning module.

## Deferred Questions

- protocol transport and schema technology
- packaging and registry ownership
- install paths and process supervision
- resource limits and OS sandbox strategy
- supported SDK languages
- community cloud services
- community web extension isolation
