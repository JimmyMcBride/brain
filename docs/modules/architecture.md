# Module Architecture

## Current Baseline

Brain remains one Go CLI with a minimal compiled-module runtime:

- `cmd/root.go` imports and registers every top-level Cobra command.
- the `App` type in `internal/app` constructs concrete workspace, notes, index,
  search, context, session, audit, skill, and module services.
- `internal/modules` owns descriptors, compiled registration, project
  configuration, local grants, lifecycle, runtime state, and health.
- `.brain/modules.yaml` stores tracked desired enablement and non-secret config.
- `.brain/state/module-grants.json` stores ignored local permission approvals.
- `internal/workspace` recognizes `AGENTS.md`, `docs/`, and `.brain/` as Brain
  knowledge and excludes local runtime state from canonical memory.
- no production module, dependency resolver, typed event bus, module audit
  service, dynamic command registration, or external extension protocol exists.

Existing services are implementation evidence, not yet stable module APIs.

## Boundary

### Brain Core owns

Project identity, durable context and memory, retrieval, context compilation,
provenance, sessions, configuration, permissions, typed events, audit primitives,
module lifecycle and capability registration, stable contracts, local runtime
foundations, and cloud-client foundations.

### Modules own

Domain models, domain workflows, domain storage adapters, domain permissions and
events, domain context selection, domain tools, and optional integrations. Modules
may request Core capabilities; they may not reach into Core's private files,
SQLite schema, concrete managers, or authentication internals.

## Stage 1 Runtime

Phase 1 adds one small package without reorganizing existing Core packages:

```text
cmd/
  modules.go                 # Core module administration
  plan.go                    # Added only when Planning reaches Phase 3
internal/
  app/                       # Composition root remains
  modules/
    descriptor.go            # API version, descriptor validation
    registry.go              # compiled-module discovery
    config.go                # tracked project configuration
    grants.go                # ignored local approvals
    runtime.go               # enablement and lifecycle
    health.go                # health vocabulary
    testmodule/              # test-only module
  official/
    planning/                # introduced incrementally after framework review
```

Current packages such as `workspace`, `search`, `session`, `projectcontext`, and
`notes` remain in place. Stable facades can be added beside them when a module
needs access; a wholesale Core package-tree move would add churn without proving
a contract.

## Phase 1 Contracts

Phase 1 intentionally implements only the contracts proven by its tests.

| Contract | Core responsibility | Module responsibility |
| --- | --- | --- |
| Descriptor | validate reverse-domain ID, semver, Brain API major, config version, declarations | declare metadata, capabilities, and exact permissions |
| Registry | reject invalid or duplicate compiled registrations | provide descriptor and factory |
| Lifecycle | gate factories, initialize once per process, report stable failures and health | validate config, initialize idempotently across processes, report health |
| Configuration | atomically persist deterministic tracked YAML and reject raw secrets | consume namespaced non-secret config |
| Permissions | atomically persist exact ignored local grants | declare every required permission |

Dependencies, dynamic module commands, events/audit, providers, jobs, migrations,
shutdown, external processes, and cloud contracts are deferred.

## Lifecycle State

Project runtime state:

```text
available -> disabled
    |
    +-> blocked
    |
    +-> enabled -> healthy | degraded | unhealthy

configured but not compiled -> unavailable
```

Enabling validates compatibility, configuration, and permissions before
initialization. Disabled, blocked, and unavailable modules are not instantiated
during inspection. Disabling preserves configuration, grants, and module-owned
data. Destructive removal is a separate future operation.

Phase 1 lifecycle:

1. Register descriptor and factories at process construction.
2. Validate Brain API/runtime compatibility.
3. Resolve project enablement.
4. Validate configuration and granted permissions.
5. Initialize eligible modules once per process.
6. Report health.

## Capability Registration Rules

- registration occurs through Core-owned registries
- collisions fail deterministically
- every capability is attributable to module ID and version
- runtime availability is explicit
- registration cannot imply permission grant
- local and cloud support are distinct
- providers receive request-scoped facades, not the concrete application object
- disabling removes active registrations without deleting durable data

## Context and Retrieval

Every module context contribution carries source identity, provenance, visibility,
revision/freshness, module identity, selection rationale, and bounded content.
Core controls budgets, deduplication, permission filtering, and final packet
assembly. Search results follow the same project and visibility rules.

Planning data is excluded when Planning is disabled. When enabled, Planning may
contribute only in response to an applicable task/request and must expose why each
artifact was selected.

## Memory Proposals

Modules may submit proposed durable-memory changes through a Core workflow.
`memory.propose`, `memory.approve`, and `memory.edit` are independent permissions.
Planning approval never implies memory approval. Direct memory writes require an
explicit policy grant and remain auditable; Planning's default is proposal-only.

## Stage 2 External Processes

Future community modules use a versioned external-process protocol. Required
properties: language independence, independent releases, compatibility
negotiation, crash containment, mediated permissions/secrets/network/filesystem,
structured logs, health, lifecycle, and safe shutdown. Connect RPC, gRPC, and
JSON-RPC remain candidates; no transport is approved here.

## Stage 3 Cloud and Web

Brain Cloud may host module services, workers, APIs, settings, tools, and events.
Core retains identity, routing, authentication, authorization, revisions, and
audit. Web navigation, project panels, editors, settings, and dashboards are
trusted future surfaces. No browser sandbox or community frontend contract is
claimed.

## Dependency Rules

Tests and import checks should enforce:

- Core never imports `internal/official/planning`
- Planning depends only on stable Core facades and its own domain/adapters
- Planning domain does not import Cobra, filesystem, GitHub, Linear, or cloud
  clients
- adapters depend inward on Planning contracts
- companion integrations cannot gain undeclared Core access
- module disablement leaves all existing Core tests and workflows functional
