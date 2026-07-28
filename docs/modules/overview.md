# Brain Modules

## Status

Phase 1 internal runtime implemented. The production binary registers zero
modules; Planning, external processes, cloud modules, and module-provided
commands remain unimplemented.

Brain is an extensible context, memory, retrieval, and workflow platform. Modules
adapt Brain to a domain or team without making that domain mandatory. Planning is
the first official optional module and the design test for the framework.

Brain Core remains useful when no modules are enabled. It owns project identity,
durable context and memory, retrieval, context compilation, provenance, sessions,
configuration, permissions, events, audit primitives, module lifecycle, stable
extension contracts, and local/cloud-client foundations.

## Module Categories

| Category | Initial runtime | Trust boundary | Examples |
| --- | --- | --- | --- |
| Brain Core | In process | Product core | projects, context, memory, retrieval, sessions |
| Official module | Compiled Go package | Trusted but interface-bound | Planning |
| Companion integration | Compiled initially; separable later | Explicit capabilities | Git, GitHub |
| Community module | External process in a future stage | Untrusted executable | company workflows, tracker adapters |

Official status does not permit private storage access or undocumented hooks.
Planning must exercise general extension contracts that an incident, architecture,
or compliance module could also use.

## Stages

1. Internal compiled modules: the registry, explicit enablement, configuration,
   permissions, declared capabilities, minimal lifecycle, health, and a
   test-only module are implemented. Events and the first production official
   module remain later work.
2. External-process community modules: versioned protocol, independent releases,
   compatibility negotiation, permission mediation, crash isolation, logs, health,
   and lifecycle control. Transport remains undecided.
3. Brain Cloud services and web extensions: module workers, APIs, events, settings,
   tools, navigation, panels, and dashboards. Frontend isolation requires a
   separate security design.

Go's native `plugin` package is not the long-term community-module mechanism.

## Enablement Contract

Modules are disabled unless explicitly enabled for a project or enabled by a
governing cloud policy. Phase 1 commands:

```text
brain modules list
brain modules show <id>
brain modules grant <id> <permission>...
brain modules revoke <id> <permission>...
brain modules enable <id>
brain modules disable <id>
brain modules health [id]
```

All commands support the root `--project` and `--json` flags.

Disabled modules:

- create no module storage
- require no module configuration
- register no effective permissions
- contribute no context or search records
- return a clear disabled-module diagnostic for direct commands
- do not prevent Brain Core workflows

Enabled modules feel native while retaining a visible module identity in
provenance, audit, permissions, configuration, and health output.

## Provisional Manifest

```yaml
id: official.planning
name: Brain Planning
version: 0.1.0
brain_api: ">=1.0 <2.0"
runtimes:
  local: true
  cloud: true
  web: future
capabilities:
  - commands
  - context_provider
  - search_provider
  - event_publisher
  - event_consumer
  - agent_tools
permissions:
  - project.context.read
  - memory.propose
  - planning.read
  - planning.brainstorm
  - planning.spec.create
  - planning.spec.edit
  - planning.spec.approve
  - planning.execute
dependencies:
  required: []
  optional:
    - official.git
    - official.github
```

Schema is provisional. A complete contract must eventually cover stable ID,
display name, version, compatible Brain API range, runtimes, capabilities,
permissions, required and optional dependencies, configuration and data schemas,
migration version, network/filesystem/secret declarations, publisher identity,
checksums, signing, and distribution source.

## Non-Goals

This architecture does not implement Planning, cloud or hybrid Planning,
community-process loading, a marketplace, frontend extensions, full GitHub
migration, immediate Plan CLI or Linear code removal, `.plan/` relocation, a
universal project-management system, source hosting, CI, or deployment.

See [Architecture](architecture.md), [Security](security.md), and
[Planning-Driven Requirements](planning-driven-requirements.md).
