---
created_at: "2026-07-29T08:41:16Z"
project: brain
slug: plan-cli-compatibility
source_brainstorm: .plan/brainstorms/plan-cli-compatibility.md
status: approved
title: Plan CLI Compatibility
type: spec
updated_at: "2026-07-29T08:50:15Z"
---

# Plan CLI Compatibility

Created: 2026-07-29T08:41:16Z

## Why

Phase 3 proved a native, optional local Planning workflow, but standalone Plan
still owns the broader local spec-first experience and remains the compatibility
source for existing users and scripts.

Phase 4 should converge those entrypoints on Brain-owned behavior without
forcing a binary cutover, importing integration-specific code, or turning
compatibility into an unbounded rewrite.

## Problem

Brain and standalone Plan expose adjacent Planning behavior without one
reviewed compatibility contract. Users cannot confidently treat the two
entrypoints as one workflow, existing automation may behave differently
depending on which command is used, and maintainers lack a bounded way to move
ownership into Brain. An unconstrained transition would also pull later
integration and legacy work into this phase.

## Goals

- Let Brain users complete a coherent local spec-first Planning workflow.
- Keep existing Plan users and automation working during a documented overlap.
- Make compatibility measurable before behavior moves between entrypoints.
- Give every standalone command an explicit transition disposition.
- Preserve open-in-place interoperability with current local workspaces.
- Establish a safe, reversible path for Brain Planning to become primary.

## Non-Goals

- Expanding remote integrations, cloud storage, or hybrid synchronization.
- Bringing Linear behavior into Brain.
- Retiring standalone Plan or publishing an end-of-support date.
- Changing the local workspace format or ownership.
- Reviving legacy epic/story creation as the target workflow.
- Adding the Brain knowledge loop.
- Designing a general community compatibility or distribution system.

## Constraints

- Brain Core remains useful with Planning disabled and imports no optional
  integration implementation.
- `brain plan` continues through module enablement and permission gates.
- Standalone `plan` works without a `.brain/` workspace and without the Brain
  binary installed.
- The supported migration source vocabulary remains `local`, `github`, and
  `hybrid`; Phase 4 implements local behavior only.
- Specs remain canonical. Execution slices are derived when execution starts
  and are not persisted as planning hierarchy.
- Shared packages import no Cobra, Brain private storage, module runtime,
  GitHub, Linear, cloud, or command-host dependencies.
- Warnings never use stdout or corrupt machine-readable output.
- Observable compatibility does not require identical package layout or
  byte-identical human help prose unless the baseline proves a script contract.
- Mutations preserve explicit confirmation, authorization, audit/event,
  atomicity, and idempotency behavior appropriate to each host.

## Solution Shape

### Conformance-first migration

Capture standalone Plan's observable behavior before moving each command family.
The compatibility manifest records:

- invocation and supported flag forms
- fixture and initial workspace state
- exit code
- stdout and stderr contract
- JSON schema and stable fields where applicable
- created, changed, unchanged, and removed paths
- confirmation and permission behavior
- result of an identical rerun

Human text is compared structurally unless an existing golden or documented
consumer requires exact text. JSON, exit classifications, file effects, and
idempotency are exact contracts unless the manifest records an intentional
versioned redesign.

### Brain-owned importable packages

Normalize Brain's root Go module to its canonical GitHub module path so
standalone Plan can pin and import it normally. Expose the smallest stable
Planning boundary needed by both hosts:

- storage-neutral domain values and validation
- application services, DTOs, ports, and deterministic render models
- schema-v3 local adapter and compatibility diagnostics
- local workflow state required by mapped commands

Do not export Cobra command trees. `brain` and `plan` retain separate presentation
adapters so root flags, help navigation, module gating, and host-specific policy
remain explicit.

The public package boundary is not a general module SDK. Export a symbol only
when both command hosts need it or a compatibility test requires the contract.

Architecture exclusions remain explicit: no subprocess delegation to the Brain
binary, nested Go module, third repository, RPC protocol, integration package,
or persisted implementation-slice format.

### Host policy

Both entrypoints call the same shared application behavior but supply different
host capabilities:

- Brain supplies module enablement, Planning permissions, audit/events, project
  root resolution, terminal/editor access, and output policy.
- Standalone Plan supplies compatibility authorization, its existing project
  root and terminal/editor behavior, and an event sink appropriate to the
  standalone host.

Standalone Plan does not require Brain module enablement. This is a deliberate
compatibility-host difference, not a fork in Planning artifact behavior.

### Mapped local workflow

Phase 4 shares and compatibility-tests these local families:

| Standalone family | Native direction | Phase 4 behavior |
| --- | --- | --- |
| `doctor` | `brain plan status` plus module health detail | shared workspace diagnostics |
| `status` | `brain plan status` | shared aggregate local status |
| `check` | `brain plan check` | shared project/spec readiness findings |
| `roadmap show/edit` | `brain plan roadmap show/edit` | shared Markdown read/replace behavior |
| `brainstorm start/show/idea/refine/challenge` | same under `brain plan brainstorm` | shared local discovery behavior |
| `brainstorm resume/switch/reopen/review/sessions/park` | same under `brain plan brainstorm` | shared guided local workflow |
| `guide current/show` | `brain plan guide current/show` | shared versioned local guide packets |
| local `discuss assess/repair/promote` | `brain plan brainstorm assess/repair/promote` | shared local maturity and direct spec promotion |
| `spec show/edit/status/analyze/checklist/initiative/execute/handoff` | same under `brain plan spec` | shared canonical spec workflow and runtime-slice derivation |

Existing Phase 3 list commands remain native conveniences even where standalone
Plan currently exposes equivalent discovery through status or filesystem-backed
queries.

Local confirmed promotion creates or updates the canonical spec directly. It
does not create an epic as an intermediate artifact. Preview actions remain
explicit and idempotent.

### Disposition-only commands

The compatibility matrix records but does not migrate these implementations:

| Standalone family | Disposition |
| --- | --- |
| `init`, `adopt`, `update` | transitional workspace migration commands; standalone until a separately approved lifecycle migration |
| `source show/set` | redesign around ownership; local read diagnostics may map to status, mutations wait for adapter/hybrid phases |
| `github ...` and remote `discuss ...` | Phase 5 GitHub adapter transition |
| `epic ...` | retire creation; preserve read/import provenance until Phase 6 compatibility cleanup |
| `story ...` | transition execution evidence; no native creation hierarchy |
| `skills ...` | redesign through Brain skill/module discovery |

Linear is not a disposition target inside Brain. A Linear-configured workspace
receives standalone migration guidance and remains Phase 6 standalone cleanup.

### Warning policy

Migration warnings begin only after a standalone command family calls the shared
implementation and passes its compatibility suite.

- warn once per process
- write only to stderr
- show only for interactive terminal use
- suppress for JSON output and non-interactive execution
- name the equivalent `brain plan` command
- provide no retirement date in Phase 4

## Flows

### Conformance capture

1. Select one mapped command case and a copied schema-v3 fixture.
2. Run the pinned standalone baseline.
3. Record normalized observable results in the manifest.
4. Run the native/shared implementation against an independent fixture copy.
5. Compare exact machine contracts and normalized human contracts.
6. Reject unexplained drift before migrating the command family.

### Native command

1. Brain resolves the registered command and verifies module enablement.
2. The host authorizes the declared Planning capability.
3. Presentation code maps flags/input into a shared request DTO.
4. The shared service validates state and calls the local adapter.
5. The host renders the shared result and emits any Brain audit/event record.
6. Repeating a confirmed idempotent mutation returns `unchanged` without a
   duplicate event.

### Compatibility command

1. Standalone Plan parses its existing command and flags.
2. Its presentation adapter maps input to the same shared request DTO.
3. The standalone host supplies compatibility authorization and host services.
4. The shared service performs the same local behavior.
5. Plan renders its preserved output contract.
6. Interactive mapped commands may emit one stderr migration warning.

### Local brainstorm promotion

1. Assess the local brainstorm and return a versioned readiness decision.
2. Draft explicit `create`, `update`, `reuse`, or `unchanged` spec actions.
3. Preview without mutation.
4. Require confirmation and host authorization for apply.
5. Write the canonical spec atomically without creating an epic.
6. Update guided-session provenance and emit the host audit/event.
7. Repeat apply returns `unchanged`.

## Data / Interfaces

- Canonical Brain Go module path and version pinned by standalone Plan.
- Public Planning domain/application/local packages with documented dependency
  rules.
- Host interfaces for authorization, audit/events, clock, terminal/editor,
  project root, and rendering policy.
- Versioned compatibility-case manifest and golden fixtures.
- Versioned guide, readiness, promotion, status, check, and execution output
  DTOs.
- Stable error classification mapped by each CLI to its compatible exit code and
  presentation.
- Schema-v3 `.plan/` files remain the shared durable artifact boundary.

Cross-repository tests pin exact revisions. Brain tests the public consumer
surface from an external-package harness. Standalone Plan pins a merged Brain
revision and runs the same fixture cases through its wrapper. Published releases
replace temporary pseudo-version pins; checked-in local `replace` directives are
not allowed.

## Risks / Open Questions

- Normalizing Brain's module path is broad mechanical churn. Isolate it in an
  execution slice with unchanged behavior and full repository verification.
- A public package can accidentally become an oversized SDK. Dependency tests
  and a small export review gate are required.
- Standalone host authorization/audit differs from Brain host policy. Tests must
  distinguish host policy from artifact behavior without silently weakening
  Brain.
- Guided-session and editor/TTY behavior is platform-sensitive. Use injected
  host interfaces and Windows fixtures rather than shell assumptions.
- Cross-repository rollout cannot land atomically. Brain's importable boundary
  lands first; Plan then pins that revision. Until the Plan PR lands, its legacy
  commands remain authoritative.
- Unknown external consumers may rely on undocumented text. Preserve exact JSON,
  exit, and file contracts; document intentional human-text changes and keep a
  rollback path.
- Phase 4 changes no `.plan/` schema and requires no workspace migration,
  backfill, or destructive conversion. Any discovered data-shape requirement
  stops this spec and returns to planning.

## Rollout

- Land Brain's module-path normalization and public package boundary without
  changing CLI behavior.
- Add the compatibility manifest and external-consumer tests.
- Migrate native and standalone command families incrementally behind passing
  contract cases.
- Land the Brain PR first, then pin its merged revision in a coordinated
  standalone Plan PR.
- Enable interactive warnings only for migrated families.
- Keep `.plan/` unchanged so rollback can return either command family to its
  prior presentation adapter.
- Rollback removes the new wrapper/package path and restores the pinned legacy
  implementation; no artifact backfill or workspace recovery is required.
- Update both repositories' docs and skills with the final mapping and overlap
  policy.
- Do not mark Phase 4 complete until both repositories pass their local,
  cross-platform, and compatibility suites.

## Verification

- Golden compatibility cases cover representative success, invalid input,
  missing workspace, future schema, unsupported legacy integration, permission
  denial, confirmation refusal, atomic failure, and idempotent rerun behavior.
- Compare supported flags, exit codes, normalized stdout/stderr, exact JSON,
  filesystem snapshots, and events for every mapped command family.
- Verify standalone Plan builds and runs without a Brain binary or `.brain/`
  workspace.
- Verify Brain Core behavior is unchanged while Planning is disabled.
- Verify local confirmed promotion creates a spec directly and never creates an
  epic or persisted slice artifact.
- Verify warnings are once-only, interactive-only, stderr-only, and absent from
  JSON/non-interactive execution.
- Test external importability with no `replace` directive.
- Run affected-package unit, race, vet, dependency, and Windows path/TTY tests.
- Run `go test ./...`, `go test -race ./...`, `go vet ./...`, and
  `go build ./...` in Brain.
- Run the standalone Plan repository's complete test, race, vet, build, and
  compatibility suite against the pinned Brain revision.
- Run `plan check --project .` and confirm no blocking or guidance findings.

## Execution Plan

At execution time, derive ordered runtime slices from this spec:

1. freeze command disposition and black-box compatibility contracts
2. normalize the Brain module path without behavior change
3. expose the narrow public Planning boundary and external-consumer tests
4. migrate workspace/query and roadmap behavior
5. migrate brainstorm/guided/local-promotion behavior
6. migrate spec readiness/execution behavior
7. switch standalone local families to shared behavior and enable warning policy
8. complete coordinated cross-repository verification, docs, and rollout

Each slice is documented by its implementation commit. Do not create permanent
slice files.

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

- [Source Brainstorm](../brainstorms/plan-cli-compatibility.md)
- [Planning CLI Migration](../../docs/planning/cli-migration.md)
- [Standalone Plan Feature Inventory](../../docs/planning/plan-feature-inventory.md)
- [Migration from Standalone Plan](../../docs/planning/migration-from-plan.md)
- [Planning Module Implementation Roadmap](../../docs/planning/implementation-roadmap.md)
- [Planning Domain Extraction](./planning-domain-extraction.md)
- [Local Planning Module](./local-planning-module.md)

## Notes

- Canonical local spec created through Plan's legacy local-promotion
  compatibility path because `plan discuss promote --apply` does not yet support
  local ownership. The generated epic artifact was intentionally removed; this
  workspace uses the spec-first model.
- Runtime execution slices will be derived only after the approved spec workflow
  starts.
