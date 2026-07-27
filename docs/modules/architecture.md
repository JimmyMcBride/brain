# Module Architecture

## Current Baseline

Brain is currently one Go CLI:

- `cmd/root.go` imports and registers every top-level Cobra command.
- the `App` type in `internal/app` constructs concrete workspace, notes, index,
  search, context, session, audit, and skill services.
- `internal/config` owns a small global configuration with no module namespace.
- `internal/workspace` recognizes `AGENTS.md`, `docs/`, and `.brain/` as Brain
  knowledge and excludes local runtime state from canonical memory.
- no module registry, permissions service, typed event bus, audit service, module
  manifest, capability negotiation, or extension protocol exists.

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

Phase 1 adds small packages without reorganizing existing Core packages:

```text
cmd/
  modules.go                 # Core module administration
  plan.go                    # Added only when Planning reaches Phase 3
internal/
  app/                       # Composition root remains
  modules/
    contract/                # API version, descriptors, capability vocabulary
    registry/                # compiled-module discovery and dependency ordering
    runtime/                 # enablement, lifecycle, health
    manifest/                # validation of built-in descriptors
    testmodule/              # trivial reference module
  official/
    planning/                # introduced incrementally after framework review
```

Current packages such as `workspace`, `search`, `session`, `projectcontext`, and
`notes` remain in place. Stable facades can be added beside them when a module
needs access; a wholesale Core package-tree move would add churn without proving
a contract.

## Candidate Contracts

Names and Go signatures are intentionally provisional. Phase 1 must derive the
smallest interfaces from tests.

| Contract | Core responsibility | Module responsibility |
| --- | --- | --- |
| Descriptor/manifest | validate ID, API range, runtime, dependencies | declare metadata and requested capabilities |
| Registrar | expose controlled registries | register commands, providers, tools, events |
| Lifecycle | order calls, persist state, report failures | validate, enable, initialize, upgrade, health, shutdown |
| Configuration | namespace, validate, migrate, resolve secrets | declare schema and defaults |
| Permission broker | approve, authorize, audit | declare and check domain permissions |
| Event bus | type/version events, enforce visibility | publish and subscribe explicitly |
| Context provider | enforce request budget and provenance | return bounded, reasoned contributions |
| Search provider | merge authorized results | expose records with source/revision data |
| Memory proposal provider | create reviewable proposals | propose; never silently write |
| Agent tool registry | validate schemas and policy | declare typed tools and mutation class |
| Job registry | retry/idempotency and health in cloud | define retry-safe cloud work |
| API registry | own routing/auth/versioning | contribute trusted versioned handlers |

Command registration must be namespaced and controlled. Core owns root flags,
help consistency, collision detection, module-disabled diagnostics, and command
policy. Module command code must not be imported by unrelated Core command
packages.

## Lifecycle State

Conceptual project state:

```text
available -> enabled -> initialized -> healthy
                |            |           |
                +---------- disabled <---+
```

Enablement and data initialization are separate. Enabling validates compatibility,
configuration, dependencies, and permissions before storage creation. Disabling
stops contributions and jobs but does not destroy data. Destructive removal is a
separate future operation.

Candidate hooks:

1. Register descriptor and factories at process construction.
2. Validate Brain API/runtime compatibility.
3. Resolve enablement and dependencies.
4. Validate configuration and granted permissions.
5. Initialize or migrate project state idempotently.
6. Register active capabilities.
7. Report health.
8. Stop jobs and shut down safely.

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
