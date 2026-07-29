---
created_at: "2026-07-29T06:50:07Z"
project: brain
slug: local-planning-module
source_brainstorm: .plan/brainstorms/local-planning-module.md
status: done
title: Local Planning Module
type: spec
updated_at: "2026-07-29T07:15:44Z"
---

# Local Planning Module

Created: 2026-07-29T06:50:07Z

## Why

Brain has approved reusable Planning behavior and an optional-module
foundation, but no Brain capability connects them to real local Planning
workspaces. Users must still leave Brain for standalone Plan.

Phase 3 should prove one useful native workflow without pulling standalone
compatibility, remote integrations, or future extension work forward.

## Problem

Brain users cannot yet enable native local Planning, read current Planning
artifacts, or make a controlled Planning change. Maintainers also cannot prove
that Planning has no effect while disabled or that Brain and standalone Plan
can safely share one workspace.

## Goals

- Make Planning an official optional Brain capability.
- Open compatible existing local Planning workspaces in place.
- Let users inspect Planning status, read brainstorms/specs, and create a
  brainstorm from Brain.
- Make every change explicit, permissioned, traceable, and safe to retry.
- Preserve offline use and sequential interoperability with standalone Plan.

## Non-Goals

- Replacing standalone Plan or matching its entire interface.
- Supporting legacy epic/story workflows.
- Adding GitHub, hybrid, cloud, or network-backed behavior.
- Supporting or importing retired Linear behavior.
- Converting existing data or changing source ownership.
- Brain memory proposals or durable Brain memory writes.
- Community extensions or generalized infrastructure.
- Persisted implementation slice artifacts.

## Constraints

- Planning remains optional and explicitly enabled per project.
- Disabled Planning cannot execute Planning commands, mutate `.plan/`, or change
  Brain Core behavior.
- `internal/planning` remains storage-neutral and imports no Cobra, filesystem,
  Brain, module-runtime, GitHub, Linear, or cloud dependencies.
- Schema-v3 file ownership and compatible serialized forms remain stable.
- Unknown future schemas and unsupported legacy integrations fail closed before
  mutation.
- Every mutation declares permission, preview/confirmation, audit/event, and
  idempotency behavior.
- Planning may consume approved read-only Brain context only through a declared
  capability.

## Solution Shape

### Official module

Register one compiled `planning` module using the Phase 1 registry and
project-scoped enablement. Declare only capabilities, permissions, and events
used by this phase. Extend module contracts only where the first official module
proves a concrete missing operation.

### Local adapter and application services

Introduce persistence ports from concrete application-service needs rather than
copying standalone Plan's manager. The adapter parses current `.plan/` files,
maps them to Phase 2 domain values, preserves narrative bodies and adapter-owned
metadata, and writes atomically with deterministic ordering.

Workspace diagnostics classify:

- missing workspace
- compatible schema-v3 local workspace
- unknown future schema
- unsupported legacy integration configuration

Only compatible local workspaces are writable. Existing compatible workspaces
open in place; this phase has no data-conversion migration.

### Native command surface

Expose the smallest useful tree:

```text
brain plan status
brain plan brainstorm list
brain plan brainstorm show <slug>
brain plan brainstorm start <title>
brain plan spec list
brain plan spec show <slug>
```

Commands call shared application services. Presentation code does not parse or
write workspace files directly. Exact standalone compatibility remains Phase 4.

### First mutation

`brain plan brainstorm start` is the sole Planning artifact mutation in this
phase. It:

1. validates module enablement and workspace compatibility
2. constructs and validates the proposed brainstorm
3. renders a preview without writing
4. requires explicit confirmation and declared permission
5. performs one atomic write
6. emits the declared audit/event record only after a real mutation
7. returns the existing equivalent result on an idempotent rerun

### Brain boundary

Planning may request approved read-only Brain context through a declared
capability. This phase does not add memory-proposal behavior and cannot write
durable Brain memory.

## Flows

### Inspect an existing workspace

1. User explicitly enables Planning for the project.
2. Planning status detects `.plan/` without mutation.
3. Adapter validates workspace schema and ownership configuration.
4. Command returns deterministic status or actionable unsupported-state
   guidance.

### Read local artifacts

1. Native command passes through the module enablement gate.
2. Application service requests artifacts from the local port.
3. Adapter maps current files into validated domain values.
4. Command renders stable text or JSON without changing files.

### Create a brainstorm

1. Native command validates enablement and compatible local ownership.
2. Application service validates title, slug, identity, and idempotency input.
3. Command previews proposed path and artifact.
4. Explicit confirmation and permission authorize the mutation.
5. Adapter atomically writes the schema-v3-compatible artifact.
6. Service emits one Planning event/audit record for a new write.
7. Repeating the same operation returns unchanged without duplicate write or
   event.

## Data / Interfaces

Exact names may follow existing package conventions. Responsibilities remain:

- `planning` module descriptor, lifecycle hooks, capabilities, permissions,
  events, and health
- Planning application services for status, brainstorm reads/creation, and spec
  reads
- minimal repositories/ports derived from those use cases
- local `.plan/` adapter records and Markdown/frontmatter mapping
- command request/response DTOs independent from persisted records
- explicit mutation preview/result carrying action, target, idempotency outcome,
  and emitted event identity

Permissions distinguish Planning reads from brainstorm creation. Events identify
the module, operation, stable artifact ID, outcome, and caller-supplied audit
context without embedding filesystem or provider clients in domain values.

## Risks / Open Questions

- Current module contracts may lack controlled namespaced command contribution.
  Add only the narrowest contract required by this compiled official module.
- `.plan/.meta/` mixes schema, guided-session, and retired integration state.
  Parse only fields required to classify compatibility; do not model Linear.
- Standalone Plan may serialize details not represented in Phase 2 domain
  values. Preserve adapter-owned narrative and metadata during the one supported
  write.
- Brain and standalone Plan may operate sequentially on the same workspace.
  Atomic writes and fixture round trips must prevent lost unrelated data.

## Rollout

- No automatic enablement.
- No automatic conversion.
- Existing Brain projects behave identically until Planning is explicitly
  enabled.
- Unsupported workspaces receive diagnostics and standalone Plan migration
  guidance.
- Phase 4 expands and compatibility-tests the native command mapping after this
  vertical path is stable.

## Verification

- Add copied standalone Plan fixtures for missing, compatible schema-v3, future
  schema, and unsupported legacy-integration states.
- Test deterministic adapter reads, preservation of adapter-owned data, atomic
  failure/retry, and standalone Plan readability after Brain writes.
- Test enabled/disabled commands, stable text/JSON output separation,
  preview/confirmation, permission denial, idempotent reruns, stable errors, and
  event/audit emission only after real mutation.
- Test module descriptor, lifecycle, health, capability, permission, and event
  declarations.
- Run `go test -race` and `go vet` on affected packages.
- Run `go test ./...` and `go build ./...`.
- Inspect dependency edges and repository search results to confirm no GitHub,
  Linear, cloud, Cobra, filesystem, Brain, or module-runtime type entered
  `internal/planning`.

## Execution Plan

- Establish official module and application contracts, including enablement,
  capabilities, permissions, events, ports, and workspace classification.
- Implement the schema-v3 local adapter with compatibility fixtures,
  deterministic reads, preservation, and atomic idempotent brainstorm writes.
- Wire the bounded native command tree and mutation controls, then complete
  compatibility, boundary, race, vet, build, and whole-repo verification.

## Analysis

### Missing Constraints

- None.

### Success Criteria Gaps

- None.

### Hidden Dependencies

- None.

### Risk Gaps

- None.

### What/Why vs How Leakage

- None.

### Recommended Revisions

- None.

## Checklist

### general

status: ok
blocking_findings: 0
guidance_findings: 0

- [ok] No findings.
## Resources

- [Source Brainstorm](../brainstorms/local-planning-module.md)
- [Planning Module Implementation Roadmap](../../docs/planning/implementation-roadmap.md)
- [Planning CLI Migration](../../docs/planning/cli-migration.md)
- [Migration from Standalone Plan](../../docs/planning/migration-from-plan.md)
- [Planning Domain Extraction](./planning-domain-extraction.md)
- [Minimal Internal Module Framework](./minimal-internal-module-framework.md)

## Notes

- Canonical local spec created through Plan's legacy local-promotion
  compatibility path because `plan discuss promote --apply` does not yet support
  local ownership. The generated epic artifact was intentionally removed; this
  workspace uses the spec-first model.
- Runtime execution produced three ephemeral slices; each implementation slice
  is documented by its branch commit and no slice file was persisted.
- Verified with focused tests, affected-package race/vet, `go test ./...`,
  `go build ./...`, `plan check`, dependency inspection, context audit, and a
  real Brain-created brainstorm read successfully by standalone Plan.
