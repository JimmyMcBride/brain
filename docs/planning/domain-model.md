---
updated: "2026-08-09T07:52:41Z"
---
# Planning Domain Model

## Status

Phase 2's pure domain values and rules are importable from
`github.com/JimmyMcBride/brain/planning` and remain stdlib-only. Phase 4 also
exposes the narrow `planning/application` service/port boundary and schema-v3
`planning/local` adapter used by both command hosts. Conformance assets and
Brain module/CLI wiring remain internal.

## Phase 2 Boundary

The implemented domain includes:

- validated artifact identities, references, and source provenance
- brainstorm, spec, initiative, and roadmap values
- closed `local`, `github`, and `hybrid` ownership modes
- Planning approval and spec lifecycle transitions
- deterministic dependency ordering, readiness, and execution queues
- runtime slice derivation and advancement after approved execution starts
- stable domain error codes

The package has no filesystem, command, module-runtime, Brain private-storage,
GitHub, Linear, or cloud dependency. It defines no repository interface;
persistence ports wait for the concrete local adapter to prove their shape.

## Aggregate Map

| Aggregate | Responsibility | Current Plan evidence | Target |
| --- | --- | --- | --- |
| Brainstorm | Discovery source and guided session | `.plan/brainstorms`, `manager.go`, guided sessions | Preserve |
| Refinement | Clarify problem, value, appetite, choices | `refinement.go` | Preserve |
| Challenge | Rabbit holes, no-gos, assumptions, simpler path | `challenge.go` | Preserve |
| Maturity assessment | Decide not-ready/repair/single/multi-spec | `collaboration.go` | Preserve, decouple source adapters |
| Promotion draft | Preview create/update/reuse/unchanged actions | collaboration/reconcile packages | Preserve reconciliation semantics |
| Spec | Canonical execution contract and status | `.plan/specs`, `manager.go` | Preserve |
| Initiative | Lightweight grouping across specs | `initiative.go`; GitHub milestone mapping | Preserve; domain identity independent of GitHub |
| Roadmap | Phase/version intent | `ROADMAP.md`, `roadmap.go` | Preserve |
| Readiness | Findings, analysis, checklists, critique | analyze/check/checklist/critique | Preserve/evolve |
| Approval | Explicit authorization to execute/publish | spec status and confirmation flags | Make first-class Planning state |
| Execution plan | Ordered queue and ephemeral slices | `spec_execute.go`, story slicing, GitHub status | Consolidate as Planning execution model |
| Runtime slice | Small execution unit; one-at-a-time discipline | `SpecExecutionSlice`, legacy stories | Preserve without mandatory tiny files |
| Handoff | Guided stage transition and execution entry | guided sessions, epic/spec handoff | Preserve/evolve |
| Planning history | Revision and workflow evidence | Markdown/history plus integration metadata | Redesign on Core audit/revisions |

## Relationships

```text
Brainstorm
  -> optional Idea Doc
  -> one Spec
  -> or Initiative containing ordered Specs
Spec
  -> readiness findings/checklists
  -> approval
  -> execution plan
  -> runtime slices
  -> verified outcome
  -> memory proposal
```

Legacy epics and stories are compatibility inputs, not target canonical
aggregates. A migration may map epic metadata to Initiative and story metadata to
runtime/external work-item records, but must preserve original content and links.

## Invariants

- Specs are canonical execution contracts.
- Brainstorms are discovery sources, not mandatory hierarchy.
- Promotion is preview-first; backend mutations require explicit confirmation.
- Reconciliation uses stable identity and explicit actions, never title guessing.
- Repeated confirmed reconciliation is idempotent.
- Planning Core uses domain identifiers and generic ports, never GitHub/Linear
  types.
- One execution slice is active at a time unless policy explicitly permits more.
- Planning approval and Brain memory approval are separate.
- Every Brain source reference retains provenance and freshness.

## Application and Adapter Boundary

The public application package now owns the repository port proven by the
schema-v3 local adapter, plus workspace diagnostics, aggregate status, local
spec checks, roadmap read/replace/parking, brainstorm capture/refinement/challenge,
guided-session state and packets, maturity/source repair, direct local spec
promotion, existing brainstorm/spec reads, and guarded mutation contracts. The
local adapter depends only on the Planning domain,
application contracts, standard library, and YAML parsing. Cobra, Brain module
runtime, GitHub, Linear, and cloud types remain outside both packages.

Future adapter responsibilities remain provisional:

- artifact repositories for brainstorm, spec, initiative, roadmap, execution
- collaboration source reader
- publication adapter
- repository change adapter
- external work-item/status adapter
- readiness check and approval provider
- Brain context requester and memory proposal sink

Do not add provider abstractions solely to preserve retired Linear code. Extend
the repository port only when a mapped command family proves a concrete shared
need.

Direct local promotion preserves the target model: a mature brainstorm becomes
one spec, or an explicit set of specs, without an epic intermediate. A confirmed
rerun reconciles by stable spec slug and source brainstorm, returning
`unchanged` or `reuse` instead of duplicating artifacts.

## Source References

Conceptual artifact metadata:

```yaml
brain_sources:
  - uri: brain://project/api/decision/token-rotation
    revision: 7
    observed_at: 2026-07-27T00:00:00Z
    purpose: architecture constraint
  - uri: brain://hive/pattern/authentication-boundaries
    revision: 12
    purpose: prior art
```

Exact schema is deferred. References must be resolvable or diagnosably stale.
