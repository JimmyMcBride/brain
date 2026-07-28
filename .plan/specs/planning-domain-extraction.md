---
created_at: "2026-07-28T21:17:36Z"
project: brain
slug: planning-domain-extraction
source_brainstorm: .plan/brainstorms/planning-domain-extraction.md
status: approved
title: Planning Domain Extraction
type: spec
updated_at: "2026-07-28T21:29:11Z"
---

# Planning Domain Extraction

Created: 2026-07-28T21:17:36Z

## Why

Planning needs one deterministic core before Brain can ship a local Planning
module or standalone Plan can become a compatibility wrapper. Extracting the
rules first lets later adapters share behavior without making the domain depend
on a particular interface, document format, integration, hosted service, or
Brain's private storage.

## Problem

The standalone Plan implementation contains valuable domain behavior, but it is
currently mixed with presentation, persistence, interactive guidance, and
integration-specific behavior. Porting it wholesale would reproduce those
dependencies inside Brain and make the module boundary nominal rather than real.

Phase 2 needs to identify and implement only the durable Planning concepts and
pure rules required by later phases. It must preserve spec-first execution and
compatibility-relevant behavior without beginning later delivery phases.

## Goals

- Establish one storage-neutral definition of Planning concepts and rules.
- Make brainstorms, specs, initiatives, roadmaps, readiness, approvals,
  execution ordering, execution plans, and runtime slices reusable by later
  Brain and standalone Plan work.
- Keep spec lifecycle, approval, dependency, readiness, and execution behavior
  deterministic.
- Create runtime execution slices only when execution of an approved spec
  starts.
- Preserve compatibility-relevant standalone Plan behavior without preserving
  its current package or storage layout.
- Limit ownership vocabulary to `local`, `github`, and `hybrid`.
- Leave Brain Core behavior unchanged when Planning is absent.

## Non-Goals

- Shipping or enabling a user-facing Planning capability.
- Changing standalone Plan behavior or existing project data.
- Implementing persistence, presentation, integrations, or hosted behavior.
- Porting guided sessions, publication, reconciliation, or collaboration-source
  orchestration.
- Supporting Linear in Brain Planning.
- Designing community extensions or a generic workflow framework.
- Creating persistent story or slice artifacts.

## Constraints

- The domain package must not import Cobra, filesystem helpers, Brain private
  managers or storage, GitHub clients, cloud SDKs, or module-runtime packages.
- Standard-library use must remain deterministic; time and identity enter as
  values rather than being read from global clocks or random generators.
- Specs are the canonical execution contract.
- Runtime slices are values returned when spec execution starts. They are not
  created during brainstorm, promotion, approval, or repository initialization.
- Domain values must not contain paths, issue numbers, GraphQL IDs, provider
  clients, Markdown frontmatter, or serialized workspace records.
- `local`, `github`, and `hybrid` are the only accepted ownership modes. There
  is no generic unknown-provider escape hatch in Phase 2.
- Invalid transitions and dependency cycles fail explicitly and do not partially
  mutate caller-owned values.
- Public behavior added in this phase must be covered by table-driven tests.
- No production composition root imports the new package in Phase 2.

## Solution Shape

### Package boundary

Add one `internal/planning` package. It owns pure values and rules only. The
package accepts complete inputs, returns new values or results, and performs no
I/O.

Do not add repository interfaces in this phase. Persistence ports should be
introduced with the Phase 3 local adapter, when a concrete caller can prove the
necessary operations. Do not copy the standalone `Manager` abstraction.

### Artifact identity and provenance

Use a validated domain identifier for stable artifact identity. Identifiers are
non-empty canonical slugs and reject surrounding whitespace or path syntax.

An artifact reference contains its kind and identifier. Optional source
references are provider-neutral values containing a URI, revision, observed
time supplied by the caller, and purpose. The domain preserves these values but
does not resolve or refresh them.

The minimum artifact models are:

- `Brainstorm`: identity, title, summary, and source references
- `Spec`: identity, title, status, approval, dependencies, verification
  requirements, and optional initiative reference
- `Initiative`: identity, title, summary, and ordered spec references
- `Roadmap`: ordered phases or entries referencing initiatives or specs

Narrative document bodies and section parsing remain adapter concerns.

### Ownership

When a use case needs to describe artifact ownership, it uses a closed
`OwnershipMode` value with exactly:

- `local`
- `github`
- `hybrid`

Ownership does not select or invoke an adapter. It is descriptive input to later
application services. Linear is not represented.

### Spec lifecycle and approval

Spec status values are:

- `draft`
- `approved`
- `implementing`
- `done`

Approval is first-class state, separate from Brain memory approval. It records a
decision and optional reason supplied by the caller; actor identity and audit
persistence are deferred to application/Core contracts.

Normal progression is:

```text
draft -> approved -> implementing -> done
```

An explicit reopen operation returns `approved`, `implementing`, or `done` to
`draft` and clears execution state. Approval is required before execution can
start. Starting an already implementing spec is idempotent for the same
execution identity; starting a draft or done spec fails.

Transition functions validate first and return a new value so invalid
transitions cannot partially update caller state.

### Readiness and dependencies

Readiness states preserve the standalone Planning vocabulary:

- `clarifying`
- `ready`
- `blocked`
- `needs_refinement`
- `done`

A typed finding has a stable code, severity, message, and optional artifact
reference. Phase 2 defines only general domain findings for invalid or incomplete
artifacts, missing approval, unresolved dependencies, and dependency cycles.
Profile-specific textual checklists remain outside the package until an adapter
can supply structured inputs.

Readiness evaluation is deterministic:

1. a completed spec is `done`
2. an unresolved dependency or cycle is `blocked`
3. an error finding or missing required contract field is `needs_refinement`
4. an explicit unresolved question is `clarifying`
5. otherwise an approved, executable spec is `ready`

Dependency evaluation uses stable spec identifiers, produces a deterministic
topological order, reports missing references, and rejects cycles with the cycle
members identified.

### Execution queue

An execution queue is a computed view over specs and dependency state. Each
entry is `ready`, `current`, `blocked`, or `done`, with stable blocking reasons.
At most one spec is current unless an explicit future policy adds parallelism.
The queue is not persisted by the domain package.

### Runtime execution slices

Execution starts from an approved spec and caller-supplied structured execution
inputs. Slice derivation uses this order:

1. valid explicit candidates supplied from the spec contract
2. up to three flow candidates supplied by an adapter
3. deterministic prepare, implement, and verify fallback slices

Each slice contains a stable slug, title, goal, ordered verification commands,
and position. Empty or placeholder candidates are ignored. Duplicate slice
identifiers and empty goals fail validation.

The execution result contains the spec reference, execution identity supplied by
the caller, ordered slices, and the active slice position. It does not create
files. Only one slice is active at a time; completing the active slice advances
to the next, and completing the final slice makes the execution complete.

Commit and pull-request traceability are application concerns. The domain
exposes stable spec, execution, and slice identities that later callers can
record in commits and PR metadata.

### Errors

Expose typed or sentinel errors with stable codes for:

- `invalid_identifier`
- `invalid_artifact`
- `invalid_ownership`
- `invalid_transition`
- `approval_required`
- `dependency_missing`
- `dependency_cycle`
- `not_ready`
- `invalid_execution`
- `invalid_slice`

Errors may carry affected artifact identifiers and blocking reasons. They must
not contain storage-provider-specific remediation.

## Flows

### Validate a planning contract

1. An adapter constructs domain values from its storage representation.
2. The domain validates identifiers, required fields, statuses, ownership, and
   references.
3. Validation returns deterministic findings without mutating the values.

### Evaluate readiness and queue order

1. The caller supplies specs and their dependencies.
2. The domain validates the dependency graph and produces a stable order.
3. Readiness combines contract findings, approval, completion, and blockers.
4. The queue exposes ready, current, blocked, and done work with reasons.

### Approve and begin execution

1. A draft spec receives an explicit Planning approval.
2. The domain returns the approved spec.
3. Beginning execution validates readiness and transitions the spec to
   implementing.
4. Runtime slices are derived at that moment from structured execution inputs.
5. No slice artifact is written.

### Advance runtime execution

1. The caller completes the active slice and supplies verification evidence.
2. The domain validates that the slice is current.
3. The execution advances to the next slice or completes.
4. The caller may record stable identities in commits and, later, the final PR.

## Data / Interfaces

Exact Go names may be refined during implementation, but the package should
remain close to these responsibilities:

```go
type ArtifactID string
type ArtifactKind string
type ArtifactRef struct { Kind ArtifactKind; ID ArtifactID }

type OwnershipMode string // local, github, hybrid only
type SpecStatus string     // draft, approved, implementing, done
type ReadinessState string
type ApprovalState string

type Brainstorm struct { /* identity, title, summary, sources */ }
type Spec struct { /* identity, status, approval, dependencies, verification */ }
type Initiative struct { /* identity, summary, ordered specs */ }
type Roadmap struct { /* ordered domain references */ }

type Finding struct { /* code, severity, message, artifact */ }
type QueueEntry struct { /* spec, state, blocking reasons */ }
type ExecutionInput struct { /* execution ID, candidates, default verification */ }
type ExecutionPlan struct { /* spec, ordered runtime slices, active position */ }
type RuntimeSlice struct { /* identity, goal, verification, position */ }
```

Prefer constructors and pure functions over mutable managers:

```go
ValidateArtifact(value any) []Finding
OrderSpecs(specs []Spec) ([]ArtifactID, error)
EvaluateReadiness(spec Spec, findings []Finding, blockers []ArtifactID) Readiness
BuildQueue(specs []Spec, current *ArtifactID) (Queue, error)
ApproveSpec(spec Spec, approval Approval) (Spec, error)
BeginExecution(spec Spec, input ExecutionInput) (Spec, ExecutionPlan, error)
CompleteSlice(plan ExecutionPlan, sliceID ArtifactID, evidence []Evidence) (ExecutionPlan, error)
ReopenSpec(spec Spec) (Spec, error)
```

Avoid a universal artifact interface, generic repository, event bus, rule engine,
or serialization framework.

## Risks / Open Questions

- Standalone Plan sometimes treats Markdown sections as behavior. The extraction
  must port the underlying rule only when it can be expressed using structured
  input; parsing remains a Phase 3 adapter concern.
- Approval and status overlap in the current implementation. Phase 2 must keep
  approval explicit without inventing identity or audit behavior that belongs
  to Brain Core.
- Queue semantics currently combine specs, legacy stories, and GitHub project
  state. Phase 2 deliberately models only spec/dependency execution; legacy and
  remote mappings remain compatibility/adapter work.
- Guided-session and promotion reconciliation state remain outside Phase 2.
  Their persistence and source ownership need concrete adapters before a
  reusable boundary is justified.
- The Brain and standalone Plan repositories cannot share an `internal` package
  directly. Phase 2 proves the API in Brain; Phase 4 decides the shared-package
  or wrapper arrangement without changing these domain semantics.
- Phase 2 changes no durable data shape and reads no existing artifacts, so it
  requires no migration, backfill, or recovery path. If rollout must be
  reversed, the unreferenced package can be removed without touching project
  data.

No product decision remains open for implementation. Exact field names may
change if tests show a smaller representation preserves the same contract.

## Rollout

1. Add pure identity, artifact, ownership, lifecycle, approval, and finding
   values with validation tests.
2. Add dependency/readiness and queue evaluation with deterministic-order and
   cycle tests.
3. Add execution-plan and runtime-slice derivation/advancement with compatibility
   vectors copied from standalone Plan behavior.
4. Document the resulting domain boundary and any intentionally deferred Plan
   behavior.

The production binary registers no Planning module and exposes no new command in
this phase. Existing projects and files are untouched. There is no data
migration or backfill; rollback removes the unreferenced package and tests.

## Verification

The implementation is accepted when:

1. The new package performs no filesystem, network, database, CLI, or module
   runtime I/O.
2. Artifact and ownership validation accepts canonical values and rejects
   whitespace, path-like identifiers, unknown kinds, and any ownership value
   other than `local`, `github`, or `hybrid`.
3. Brainstorm, spec, initiative, and roadmap values validate required identity
   and reference invariants.
4. Spec transitions follow the documented progression and explicit reopen path.
5. Execution cannot begin without Planning approval and executable readiness.
6. Dependency ordering is deterministic, reports missing references, and
   identifies cycles.
7. Readiness precedence matches done, blocked, needs-refinement, clarifying,
   then ready.
8. Queue results are stable regardless of input map or slice order and allow at
   most one current spec.
9. No runtime slices exist before `BeginExecution`.
10. Execution prefers valid explicit candidates, then at most three flow
    candidates, then deterministic prepare/implement/verify fallbacks.
11. Empty placeholders are ignored and invalid or duplicate slices fail
    explicitly.
12. Only the active slice can complete, verification evidence is retained as a
    value, and completion advances exactly one position.
13. Completing the final slice marks the execution complete without persisting
    anything.
14. Compatibility-vector tests cover relevant standalone Plan execution,
    readiness, status, initiative, roadmap, and dependency behavior without
    copying storage concerns.
15. No production composition root imports or registers Planning.

Required repository verification:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`

## Execution Plan

- [ ] Define execution slices when implementation begins

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

- [Source brainstorm](../brainstorms/planning-domain-extraction.md)
- `docs/planning/domain-model.md`
- `docs/planning/plan-feature-inventory.md`
- `docs/planning/implementation-roadmap.md`
- `docs/planning/storage-and-modes.md`
- `docs/modules/planning-driven-requirements.md`
- `JimmyMcBride/plan` `develop`, especially `internal/planning` and tests

## Notes

This spec intentionally starts the Planning domain only. Runtime execution
slices are derived after this spec is approved and execution begins; they are
not separate planning artifacts committed in advance.
