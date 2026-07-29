# Brain Modules

## Status

Phase 1 internal runtime and Phase 3's bounded local Planning module are
implemented. The production binary registers `official.planning`, disabled by
default. External processes, cloud modules, remote Planning adapters, and full
Plan command compatibility remain unimplemented.

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

1. Internal compiled modules: registry, explicit enablement, configuration,
   permissions, declared capabilities/commands/events, minimal lifecycle,
   health, a test module, and the first production official module are
   implemented.
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

## Current Planning Descriptor

```yaml
id: official.planning
name: Brain Planning
version: 0.1.0
brain_api_major: 1
config_version: 1
capabilities:
  - commands
  - context.read
  - events.publish
  - storage.local-plan
permissions:
  - project.context.read
  - planning.read
  - planning.brainstorm
commands:
  - plan
events:
  - planning.brainstorm.created
```

The descriptor declares only Phase 3 behavior. Memory, spec mutation/execution,
integrations, cloud, agent tools, and community distribution remain future
contracts.

## Non-Goals

This architecture does not implement Planning, cloud or hybrid Planning,
community-process loading, a marketplace, frontend extensions, full GitHub
migration, immediate Plan CLI or Linear code removal, `.plan/` relocation, a
universal project-management system, source hosting, CI, or deployment.

See [Architecture](architecture.md), [Security](security.md), and
[Planning-Driven Requirements](planning-driven-requirements.md).
