---
created_at: "2026-07-28T05:00:28Z"
project: brain
slug: minimal-internal-module-framework
status: approved
title: Minimal Internal Module Framework
type: spec
updated_at: "2026-07-28T05:02:03Z"
---

# Minimal Internal Module Framework

Created: 2026-07-28T05:00:28Z

## Why

Brain needs a small, proven extension seam before Planning or any other domain is
moved into the product. The first framework slice must prove that optional
compiled modules can be described, configured, permissioned, initialized, and
observed without changing the behavior of Brain Core when no modules are
enabled.

## Problem

Brain currently has architecture decisions for official and future community
modules, but it does not have an executable internal module contract. Moving
Planning first would couple framework design to one large domain and make it
harder to prove that Brain remains useful by itself.

Phase 1 therefore needs the smallest internal framework that can answer:

- Which compiled modules are available?
- Which modules does this project intend to enable?
- Has the user granted every permission the enabled module declares?
- Is the module compatible, valid, initialized, and healthy?
- Can all of this remain a no-op when the production binary registers no
  modules?

## Goals

- Define and validate compiled module descriptors.
- Register module factories without instantiating disabled modules.
- Persist desired enablement and non-secret configuration in a tracked project
  file.
- Persist explicit permission grants in an ignored local file.
- Provide predictable lifecycle and health behavior.
- Expose a small `brain modules` CLI with stable human and JSON output.
- Prove the framework through an injected test-only module.
- Preserve all existing Brain behavior when no modules are registered or
  enabled.

## Non-Goals

- Implementing or importing Planning.
- Module dependency declaration or resolution.
- Module-defined dynamic CLI commands.
- Events, audit trails, jobs, shutdown hooks, upgrades, or migrations.
- Storage, context, search, memory, tool-provider, network, or secrets
  facilities for modules.
- External-process or community-module protocols.
- Cloud or hybrid module execution.
- A generic module configuration editor.
- Production reference modules.

## Constraints

- Brain Core must remain fully useful with zero modules.
- Official modules are compiled Go packages; future community modules remain a
  separate external-process design problem.
- Disabled and blocked modules must not be instantiated and must run no
  lifecycle hooks.
- Read-only module commands must create no files.
- The first mutation creates only the file it needs and writes atomically.
- Tracked configuration must contain no raw secrets; future secret values may be
  referenced only.
- Missing configuration and grant files mean zero modules enabled and zero
  permissions granted.
- Unknown module configuration must be preserved so a temporarily unavailable
  binary does not destroy project intent.
- Configuration remains after disablement, and local grants remain until
  explicitly revoked.
- Existing Brain commands, startup, tests, and build output must remain
  unchanged.

## Solution Shape

### Package boundary

Start with one `internal/modules` package:

- `descriptor.go`
- `registry.go`
- `config.go`
- `grants.go`
- `runtime.go`
- `health.go`
- `testmodule/` for tests only

Add `cmd/modules.go` for CLI integration. Do not create a fake production
module.

### Descriptor and registry

Each registration contains a descriptor and factory. A descriptor contains:

- `ID`: stable reverse-domain identifier
- `Name`: human-readable name
- `Version`: semantic version
- `BrainAPIMajor`: supported Brain module API major
- `ConfigVersion`: integer configuration schema version
- `Capabilities`: declarations for inspection only
- `Permissions`: exact permissions required to enable the module

Descriptor validation rejects invalid identifiers or versions, unsupported API
majors, and duplicate IDs. Capabilities never imply or automatically grant
permissions.

The registry stores descriptors and factories. The runtime instantiates only
modules that are enabled, compatible, validly configured, and fully granted.

### Lifecycle

The minimal module interface provides:

- `Validate(context.Context, ModuleContext, Config) error`
- `Initialize(context.Context, ModuleContext, Config) error`
- `Health(context.Context, ModuleContext) Health`

`ModuleContext` initially exposes only `Project ProjectRef`. It does not expose
the application object, SQLite, a state path, logger, network client, secrets,
or other Brain internals.

Validation is pure and read-only. Initialization runs once per process and must
be safe to repeat across processes. Health runs only after successful
initialization.

Enablement and disablement belong to the runtime, not module hooks. Defer
`Enable`, `Disable`, `Upgrade`, `Migrate`, and `Shutdown` hooks.

### Runtime state

Module state is one of:

- `available`: compiled into the binary but absent from project configuration
- `disabled`: known module explicitly configured as disabled
- `enabled`: enabled, compatible, granted, valid, and initialized
- `blocked`: desired enabled state cannot run because a prerequisite is missing
- `unavailable`: project configuration exists for a module not compiled into
  this binary

Initialized modules separately report `healthy`, `degraded`, or `unhealthy`.
Stable failure codes are:

- `module_unavailable`
- `api_incompatible`
- `config_invalid`
- `permission_missing`
- `initialize_failed`

### Composition

Add `ModuleRegistrations []modules.Registration` to `app.Options`. The
application builds the runtime and exposes it to the command layer. Core
commands import only `internal/modules`; future official modules are wired at
the outer composition root.

The production binary registers zero modules in Phase 1. Tests inject the
test-only module. No package in core, app, or modules imports Planning.

## Flows

### Inspect

1. `brain modules list` loads registrations, project configuration, and local
   grants without writing.
2. Known modules report available, disabled, enabled, or blocked.
3. Configured unknown modules report unavailable.
4. `brain modules show <id>` reports descriptor, desired state, configuration
   version, declared permissions and capabilities, grants, runtime state, and
   health when applicable.

### Grant and enable

1. `brain modules grant <id> <permission>...` validates requested permissions
   against the descriptor and atomically updates only local grants.
2. `brain modules enable <id>` validates the descriptor, API compatibility,
   configuration version and contents, and exact required grants.
3. Missing grants leave the module disabled and print the exact `brain modules
   grant` command needed.
4. Successful validation persists desired enablement, instantiates the module,
   initializes it, and reports enabled state.

### Disable and revoke

1. `brain modules disable <id>` persists disabled state without removing
   configuration, grants, or module-owned data.
2. Disable also works for a configured but unavailable module.
3. `brain modules revoke <id> <permission>...` removes local grants, including
   stale grants for an unavailable module.
4. Revoking a required permission from a desired-enabled module makes it
   blocked; it is not instantiated.

### Health

`brain modules health [id]` initializes only eligible desired-enabled modules,
then reports their health. Disabled, blocked, and unavailable modules run no
module code.

## Data / Interfaces

### Tracked project configuration

`.brain/modules.yaml`:

```yaml
schema_version: 1
modules:
  com.example.module:
    enabled: true
    config_version: 1
    config: {}
```

The YAML serializer is deterministic. Disabled entries retain their
configuration. Unknown entries round-trip unchanged. The file is written
atomically with normal tracked configuration permissions (`0644` on Unix).

### Local permission grants

`.brain/state/module-grants.json` stores exact granted permission strings by
module ID. It is ignored by Git and written atomically with `0600` permissions
on Unix and best-effort user-only ACLs on Windows.

A module version that adds a required permission becomes blocked until the new
permission is explicitly granted. Grants never expand automatically.

### CLI

Phase 1 commands:

- `brain modules list`
- `brain modules show <id>`
- `brain modules grant <id> <permission>...`
- `brain modules revoke <id> <permission>...`
- `brain modules enable <id>`
- `brain modules disable <id>`
- `brain modules health [id]`

Every command supports stable JSON output in addition to concise human output.
There is no generic `config set` command; project configuration is edited as
human-readable YAML.

## Risks / Open Questions

- Exact Go type names and output field names may be refined during
  implementation, but they must preserve the behavior and stable error codes in
  this spec.
- Windows ACL enforcement is best effort in Phase 1 and must be documented and
  tested where the platform permits.
- Module configuration validation needs a narrow contract that avoids adopting
  a schema framework prematurely.
- Initialization failure must not leave desired configuration partly written or
  make later read-only inspection mutate the project.

There are no open product decisions blocking implementation. Module
dependencies are explicitly deferred rather than represented in the Phase 1
descriptor.

## Rollout

1. Add descriptor, registration, runtime, and test-module primitives without
   wiring a production module.
2. Add tracked configuration and local grant stores with atomic persistence.
3. Add CLI commands and integration coverage.
4. Keep the production registration list empty until a separately approved
   official-module spec is ready.

No migration is required. Existing projects without the new files behave as
zero-module projects.

## Verification

The implementation is accepted when:

1. Listing modules in an existing repository succeeds, returns an empty result,
   and writes nothing.
2. An injected test module appears as available and can be explicitly disabled.
3. Duplicate registrations and invalid descriptors fail deterministically.
4. Missing, invalid, or API-incompatible configuration blocks enablement.
5. Missing permission output includes the exact grant command.
6. Grant followed by enable produces enabled and healthy state.
7. Disabled and blocked modules are never instantiated and run no hooks.
8. Revoking a required grant changes a desired-enabled module to blocked.
9. Disabling preserves configuration, grants, and module-owned data.
10. Unknown tracked modules report unavailable and can be disabled.
11. Unknown stale grants can be revoked.
12. Configuration and grant writes are atomic.
13. Human and JSON output remain stable under tests.
14. Existing Brain behavior and commands are unchanged.
15. Grant files use `0600` on Unix and best-effort user-only ACLs on Windows.
16. No module files are created before the first relevant mutation.

Required repository verification:

- `go test ./...`
- `go build ./...`
- race-sensitive module tests in CI where supported

## Execution Plan

### Slice 1: Descriptor, registry, and runtime

- Add descriptor validation, registration, runtime state, lifecycle contracts,
  and test-only module injection.
- Verify duplicate rejection, API compatibility, non-instantiation guarantees,
  initialization behavior, and health behavior.

### Slice 2: Configuration and grants

- Add deterministic tracked YAML configuration and local JSON grant stores.
- Verify no-file defaults, unknown-entry preservation, permission expansion,
  file modes, and atomic writes.

### Slice 3: CLI and integration

- Add the seven `brain modules` commands with human and JSON contracts.
- Verify all acceptance scenarios plus unchanged existing Brain test and build
  behavior.

Each slice should be reviewable independently, but all slices belong to this
single Phase 1 spec.

## Analysis

### Missing Constraints

- None.

### Success Criteria Gaps

- None.

### Hidden Dependencies

- [warn] The spec appears to depend on external interfaces, but the dependency risk is not named explicitly.

### Risk Gaps

- None.

### What/Why vs How Leakage

- [warn] The narrative sections include implementation detail that belongs in Solution Shape or Data / Interfaces.

### Recommended Revisions

- [warn] Call out integration contracts, failure handling, and ownership risks under ## Risks / Open Questions.
- [warn] Keep ## Why, ## Problem, ## Goals, and ## Non-Goals product-facing, then move technical detail into ## Solution Shape or ## Data / Interfaces.

## Checklist

### general

status: ok
blocking_findings: 0
guidance_findings: 0

- [ok] No additional general-checklist findings. The advisory analysis warnings
  above remain non-blocking.

## Resources

- `.plan/brainstorms/integrate-planning-as-an-official-brain-module.md`
- `.plan/PROJECT.md`
- `.plan/ROADMAP.md`
- `docs/modules/overview.md`
- `docs/modules/architecture.md`
- `docs/modules/planning-driven-requirements.md`
- `docs/planning/implementation-roadmap.md`
- `docs/adr/0002-official-and-community-module-model.md`
- `docs/adr/0011-planning-and-memory-permissions-are-separate.md`
- `https://github.com/JimmyMcBride/plan`

## Notes

- Reviewed collaboratively and approved as the Phase 1 implementation contract
  on 2026-07-27.
- Planning remains unimplemented until this framework lands and is reviewed.
