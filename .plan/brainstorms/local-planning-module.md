---
created_at: "2026-07-29T05:01:45Z"
epic: local-planning-module
project: brain
slug: local-planning-module
spec: local-planning-module
status: promoted
title: Local Planning module
type: brainstorm
updated_at: "2026-07-29T06:50:07Z"
---

# Brainstorm: Local Planning module

Started: 2026-07-29T05:01:45Z

## Focus Question

Define Phase 3's official local Planning module: wire the existing storage-neutral Planning domain through the compiled module runtime, preserve local .plan/ compatibility, expose native Brain Planning commands, and add explicit permissions/events without importing GitHub, Linear, or cloud work.
## Desired Outcome

An approved, bounded Phase 3 spec for Brain's optional local Planning module.

## Vision

A user can enable Planning in Brain and use an existing local `.plan/`
workspace through native `brain plan` commands, while Brain remains unchanged
when Planning is disabled. Phase 3 proves local storage, module wiring,
permissions, and events; full standalone CLI compatibility remains Phase 4.

## Supporting Material

- [Planning Module Implementation Roadmap](../../docs/planning/implementation-roadmap.md)
- [Planning CLI Migration](../../docs/planning/cli-migration.md)
- [Migration from Standalone Plan](../../docs/planning/migration-from-plan.md)
- [Planning Domain Extraction spec](../specs/planning-domain-extraction.md)
- [Minimal Internal Module Framework spec](../specs/minimal-internal-module-framework.md)

## Constraints

- Planning remains optional; disabled projects gain no commands, writes, or changed Core behavior.
- Retain existing .plan/ schema v3 and file ownership; unknown future schemas fail closed.
- Phase 3 implements local storage only. GitHub/hybrid adapters remain later work; Linear never enters Brain.
- Keep internal/planning storage-neutral and use formal module/application contracts.
- Every mutation needs declared permission, confirmation, audit/event behavior, and idempotency.
- Planning may read approved Brain context but must not silently write durable Brain memory.
- Full standalone plan command, flag, JSON, and exit-code compatibility remains Phase 4.

## Open Questions

- None. The refinement decision snapshot resolves the command surface, first
  mutation, workspace detection, and explicit enablement boundaries.
## Ideas

- Implement the official compiled Planning module using the existing module runtime and internal/planning domain contracts.
- Retain .plan/ schema v3 compatibility, deterministic local reads and writes, workspace detection, preview-first migration, and fail-closed handling for unknown future schemas.
- Expose native brain plan commands through shared application services while leaving standalone Plan CLI compatibility work to Phase 4.
- Declare Planning permissions, events, and read-only Brain context access; Planning must not silently write durable Brain memory.
## Raw Notes

## Refinement

### Problem

Brain has approved Planning domain contracts and module infrastructure, but no optional module connects them to real local .plan/ workspaces. Users must still leave Brain and use standalone Plan; Brain cannot yet prove local compatibility, disabled-state isolation, or permissioned Planning mutations.

### User / Value

AI agents and developers using Brain can enable Planning per project, open existing local .plan/ workspaces without conversion, and run a coherent native workflow. They gain one project context surface while preserving offline use, predictable files, and standalone Plan overlap during migration.

### Appetite

One bounded end-to-end Phase 3 spec: enable the module, detect and open compatible local workspaces in place, exercise deterministic reads and permissioned writes through a concrete adapter, register a minimal coherent brain plan command surface, and prove events/disabled-state isolation. Exclude full CLI parity, legacy command coverage, GitHub/hybrid behavior, and any destructive data conversion.

### Remaining Open Questions

- None.

### Candidate Approaches

- Recommended: thin vertical workflow. Explicitly enable Planning; passively detect .plan/; expose native status plus bounded brainstorm/spec reads and brainstorm creation; route all behavior through shared application services and the local adapter. Use brainstorm creation as the first permissioned, confirmed, evented, idempotent write.
- Adapter-first alternative: ship workspace detection, health, fixture reads, and migration preview with almost no authoring commands. Smaller, but weak user value and does not convincingly prove normal local writes.
- Broad-facade alternative: port every standalone Plan local command now. Strong immediate parity, but collapses Phase 3 and Phase 4 and exceeds appetite.

### Decision Snapshot

Choose the thin vertical workflow. Detect existing .plan/ automatically but require explicit Planning enablement. Phase 3 exposes status, bounded brainstorm/spec reads, and brainstorm creation through shared application services plus a concrete local adapter. Brainstorm creation is the first permissioned, previewed/confirmed, evented, idempotent mutation. Full CLI parity, legacy coverage, GitHub/hybrid adapters, Linear, cloud, and destructive conversion remain out of scope.

## Challenge

### Rabbit Holes

- Copying standalone Plan packages wholesale instead of implementing the approved ports.
- Expanding the native command tree toward full Plan compatibility.
- Generalizing module APIs or storage abstractions for hypothetical community/cloud adapters.
- Redesigning .plan/ schema or converting current workspaces when open-in-place compatibility is enough.
- Pulling GitHub, hybrid sync, Brain memory proposals, legacy epic/story behavior, or Linear cleanup into Phase 3.

### No-Gos

- No Planning command execution or .plan/ mutation while module is disabled.
- No Linear schema, adapter, command, permission, event, import, or compatibility type.
- No GitHub/hybrid/cloud adapter, network dependency, or hosted-service requirement.
- No silent Brain memory write or source-of-truth ownership change.
- No write against unknown future schemas and no destructive migration.
- No persisted implementation slices during shaping; runtime slices begin only when approved spec execution starts.

### Assumptions

- Current .plan/ schema v3 and representative standalone Plan fixtures are stable enough to define compatibility.
- Phase 2 domain contracts cover required semantics; Phase 3 may add application services and adapter DTOs without contaminating the domain.
- The module runtime can support controlled namespaced command registration with at most a narrow contract extension.
- Local files remain authoritative and atomic deterministic writes can preserve sequential interoperability with standalone Plan.
- Brainstorm creation is low-risk and representative enough to prove the mutation contract.

### Likely Overengineering

Building a universal adapter SDK, generic command contribution framework, durable event bus, or multi-version migration engine before one official local module needs them. Phase 3 should use the smallest formal contracts that support one compiled module, current schema compatibility, synchronous observable events/audit, and one vertical authoring workflow.

### Simpler Alternative

One compiled Planning module, one current-schema local adapter, passive workspace diagnostics, and a narrow native tree: status; brainstorm list/show/start; spec list/show. Only brainstorm start mutates. Existing workspaces open in place; unsupported/Linear/future-schema workspaces receive read-only diagnostics and guidance. Fixture, disabled-state, permission, event, atomic-write, and idempotency tests prove the boundary.

## Promotion map

### Spec 1 — Local Planning module

Connect the approved Planning domain to Brain's compiled module runtime through
one concrete local `.plan/` adapter and the smallest useful native command
surface.

Scope:

Register the official optional Planning module; add current-schema workspace
detection and local read/write application services; expose `status`,
`brainstorm list/show/start`, and `spec list/show`; enforce enablement,
permissions, confirmation, events/audit, atomic writes, and idempotency. Open
compatible workspaces in place. Diagnose missing, future-schema, and unsupported
legacy integration states without mutation.

Acceptance criteria:

- Planning is explicitly enabled per project; disabled-state tests prove no Planning command execution, `.plan/` mutation, or changed Brain Core behavior.
- The local adapter reads representative standalone Plan schema-v3 fixtures into the storage-neutral domain with deterministic ordering and no GitHub, Linear, cloud, Cobra, or Brain dependency in `internal/planning`.
- Workspace diagnostics distinguish missing, compatible, unknown future-schema, and unsupported legacy-integration states; only compatible local workspaces are writable.
- Native `brain plan` status, brainstorm list/show/start, and spec list/show commands call shared application services rather than filesystem code directly.
- Brainstorm creation previews before confirmation, checks a declared permission, writes atomically, emits the declared Planning event/audit record only for a real mutation, and produces no duplicate on an idempotent rerun.
- Planning may consume approved read-only Brain context through a declared capability but cannot write durable Brain memory.
- Existing compatible `.plan/` workspaces remain usable by standalone Plan after Brain Planning reads or writes them.
- Full standalone CLI compatibility, legacy epic/story commands, GitHub/hybrid/cloud adapters, Linear support, destructive conversion, and persisted shaping-time slices remain excluded.

Verification:

- Run focused local-adapter, application-service, module, permission/event, and CLI tests against copied standalone Plan fixtures for every supported diagnostic state.
- Run command tests for enabled/disabled behavior, text and JSON separation, preview/confirmation, permission denial, idempotent reruns, event emission, and stable errors.
- Run atomic-write failure/retry tests and verify standalone Plan can still read Brain-written fixtures.
- Run `go test -race` on affected packages, `go vet` on affected packages, `go test ./...`, and `go build ./...`.
- Inspect dependencies and repository search results to confirm no GitHub, Linear, cloud, Cobra, or Brain type entered the storage-neutral domain.

Dependencies: none.

Readiness: ready.
