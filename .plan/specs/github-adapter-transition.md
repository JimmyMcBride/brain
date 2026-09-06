---
created_at: "2026-09-02T14:22:24Z"
project: brain
slug: github-adapter-transition
source_brainstorm: .plan/brainstorms/github-adapter-transition.md
status: approved
title: GitHub Adapter Transition
type: spec
updated_at: "2026-09-02T14:34:17Z"
---

# GitHub Adapter Transition

Created: 2026-09-02T14:22:24Z

## Why

Teams already shape plans in GitHub Discussions and track delivery through
Issues, Projects, and pull requests. Brain should become their primary Planning
entrypoint without forcing abandonment of those workflows, while projects that
plan locally remain independent of GitHub.

## Problem

GitHub-backed planning works only through standalone Plan today, splitting the
experience and ownership between two entrypoints. Its current behavior is also
shaped around one provider, making it hard to preserve safely inside Brain
without turning a transitional integration into the permanent Planning model.
An unbounded move risks breaking existing workflows, weakening local-first use,
or carrying retired hierarchy into Brain.

## Goals

- Preserve the current collaborative planning, publication, adoption, delivery
  tracking, repository evidence, and reconciliation outcomes.
- Give Brain users a native GitHub-backed workflow while standalone Plan and
  existing automation remain compatible.
- Keep current local, GitHub, and hybrid workspaces usable in place.
- Preserve preview, confirmation, authorization, audit, durable identity,
  explicit actions, recoverable failure, and safe reruns.
- Keep GitHub optional so local Planning remains deterministic and offline.
- Keep specs canonical and external tracking subordinate to Planning intent.
- Document a clear transitional support posture without premature retirement.

## Non-Goals

- Building Cloud Planning or generalized hybrid synchronization.
- Bringing Linear into Brain.
- Designing community-module distribution or a general provider marketplace.
- Making GitHub objects canonical Planning artifacts.
- Adding a new epic/story hierarchy or persisting implementation slices.
- Moving or converting existing Planning artifacts and integration mappings.
- Expanding GitHub behavior beyond the retained compatibility baseline.
- Retiring GitHub source mode or publishing a removal date.

## Constraints

- Brain Core and local Planning remain useful with no GitHub account, gh
  executable, network access, adapter configuration, or GitHub metadata.
- Planning domain, planning/application, and planning/local expose no GitHub
  imports or API-shaped types. Generic ports model only proven Planning needs.
- GitHub is an optional compiled official adapter. Phase 5 does not design the
  future external community-module protocol.
- Existing local, github, and hybrid values remain readable. Phase 5 preserves
  current split ownership but does not design Phase 9 hybrid sync.
- .plan/.meta/github.json remains readable in place and retains its current
  observable shape during compatibility overlap; no backfill is required.
- Specs remain canonical. Initiative issues, milestones, Project items, and
  legacy stories are external tracking or compatibility records.
- Mutations stay preview-first, explicitly confirmed, permissioned, audited,
  action-classified, and idempotent. Ambiguous identity fails closed.
- GitHub credentials remain owned by gh; Brain and Plan persist no token.
- Linear, cloud storage, generalized hybrid synchronization, Brain knowledge
  proposals, legacy story creation, and source-mode retirement are excluded.

## Solution Shape

### Conformance-first baseline

Pin standalone Plan v0.1.29 at 898b2a4c4703 in a versioned manifest before
moving behavior. Capture invocation, flags, fixture and fake-provider state,
exit code, stdout/stderr and JSON, provider-call transcript, created/updated
remote identities, GitHub metadata effects, confirmation, partial failure, and
an identical rerun.

Required command families are remote discuss assess/repair/promote, github
enable/adopt/reconcile/project status, GitHub-aware check/guide behavior, and
repository/planning-PR evidence used by reconciliation. GitHub-backed story
creation remains a compatibility-only Plan surface; capture only shared adapter
primitives and evidence needed to keep it working.

Required CI uses injected fakes and fixtures. Live GitHub smoke checks are
optional maintainer verification and never create required test resources.

### Provider-neutral application boundary

Extend planning/application only with types and ports proven by the manifest:

- collaboration source: content, contributions, canonical source reference,
  opaque revision, and guarded repair
- publication target: inspect and apply initiative/spec publication actions,
  semantic grouping, dependencies, relationships, and readiness
- repository evidence: repository identity, default/current ref, commit, and
  planning change-request state
- external mapping repository: stable artifact/source/group/workspace references
  and last reconciliation evidence
- execution workspace: attach published work, inspect supported fields/status,
  and apply a Planning execution status

Provider-neutral external references contain provider, kind, opaque ID, display
ID, URL, and revision. They do not expose issue numbers, node IDs, milestones,
labels, Project field IDs, GraphQL, or gh arguments as Planning concepts.

Application services own assessment, source repair, deterministic publication
planning, confirmed publication, adoption, reconciliation, and execution-status
transitions. Publication actions remain create, update, reuse, or unchanged.
Ports perform provider operations and return typed evidence; they do not own
Planning maturity, spec split, readiness, or action classification.

### Official GitHub adapter

Add importable planning/github, depending inward on Planning domain and
application contracts. It owns:

- injected, context-aware gh execution and GitHub error translation
- repository, Discussion, issue, label, milestone, sub-issue, blocked-by,
  Project, field, item, and status API values
- managed issue-body markers and current Plan label/field conventions
- mapping between semantic Planning roles and GitHub objects
- exact read/write compatibility for current .plan/.meta/github.json
- GitHub-specific health, pagination/rate diagnostics, and safe drift repair

Adapter stores no credential; gh auth remains authoritative. Existing
single-select Project options remain manual remediation because GitHub
regenerates option IDs. Adapter API is not a general GitHub SDK.

### Host policy and enablement

Brain Planning module configuration version 2 accepts one explicit GitHub
adapter enablement value. Enablement is separate from artifact ownership:
existing github and hybrid modes are compatibility inputs, and a local
workspace may use a Discussion as a collaboration source.

brain plan github enable --confirm preflights gh, records enablement, and
initializes compatible adapter state without changing source ownership.
brain plan github disable --confirm stops GitHub commands/network behavior and
preserves artifacts and mappings. Disabled adapter means no gh lookup, network
request, GitHub metadata mutation, or GitHub event. Adapter health can degrade
independently without making local Planning unhealthy.

Remote reads require planning.read. Discussion repair requires new
planning.collaboration. Publication, adoption, reconciliation, and configuration
mutations require new planning.publish. Project execution status retains
planning.execute. Every real mutation emits one typed event and Brain audit
record; previews, denials, pre-mutation failures, and unchanged reruns emit none.

### CLI mapping

Native remote collaboration maps to:

~~~text
brain plan brainstorm assess --discussion <number-or-url>
brain plan brainstorm repair --discussion <number-or-url> --spec <title> --confirm
brain plan brainstorm promote --discussion <number-or-url> [--apply --confirm]
~~~

Provider administration remains explicit:

~~~text
brain plan github enable|disable
brain plan github adopt
brain plan github reconcile
brain plan github project status
~~~

Standalone Plan retains command names, flags, JSON, and fallback behavior as
presentation wrappers. Phase 5 adds no native brain plan source set and no
native story creation. Existing standalone source and GitHub-backed story
commands remain documented compatibility surfaces.

### Failure and idempotency policy

Ambiguous identity fails before mutation; title alone is never sufficient.
Partial remote failure does not trigger destructive rollback. Results identify
completed actions, failing action, durable evidence, and whether manual fallback
is allowed. Reruns re-read remote state and mappings, then classify actions as
update, reuse, or unchanged instead of duplicating objects.

Manual gh mutation remains forbidden unless the typed result explicitly sets
manual_fallback_allowed=true. Adapter-state writes are atomic locally. A state
write failure after remote mutation remains recoverable through remote identity
and managed markers on the next preview.

## Flows

### Remote assessment and repair

1. Host verifies Planning module, adapter enablement, and read permission.
2. Adapter resolves Discussion content, contributions, reference, and revision.
3. Shared application behavior returns the same versioned maturity decision used
   for local sources.
4. Repair previews canonical Specs content, checks planning.collaboration,
   requires confirmation, and guards source revision before updating.
5. Real repair emits audit/event evidence; conflict or unchanged body does not.

### Publication and adoption

1. Shared assessment chooses single spec or initiative plus specs.
2. Application loads stable mappings and provider candidates; ambiguity fails.
3. Application produces deterministic action/relationship/group/workspace plan.
4. Host checks planning.publish and confirmation.
5. Adapter applies ordered actions and returns opaque evidence after each.
6. Mapping state persists atomically; result reports all action classifications.
7. Confirmed rerun re-inspects remote state and performs no duplicate mutation.

### Repository reconciliation

1. Adapter reports current repository/ref/commit and planning change request.
2. Application determines changes to document references, readiness,
   relationships, issue bodies, or Project fields.
3. Preview reports safe updates and manual-only drift.
4. Confirmed reconcile applies classified safe changes, updates evidence, and
   emits one audit/event record only when state changed.

### Execution status

1. Application resolves tracked spec/initiative and execution workspace.
2. Adapter validates required fields, options, and item membership.
3. Host checks planning.execute and confirmation.
4. Adapter applies todo, in-progress, in-review, or done. Missing options remain
   explicit manual remediation.

### Disablement

1. User previews and confirms GitHub adapter disablement.
2. Brain retains .plan and GitHub mapping state unchanged.
3. GitHub commands return adapter-disabled diagnostic; local commands remain
   healthy and perform no network work.

## Data / Interfaces

- Versioned application DTOs for collaboration source, maturity result,
  publication plan/result, adoption result, reconciliation plan/result,
  repository evidence, external reference, execution workspace/status, and
  typed integration errors.
- Capability-sized ports for collaboration source, publication, repository
  evidence, external mappings, and execution workspace.
- Error classes: adapter disabled, provider unavailable, unauthenticated,
  unauthorized, revision conflict, ambiguous identity, unsupported capability,
  partial failure, and manual remediation required.
- Planning module config version 2 with explicit GitHub enablement; no token,
  repository ID, source ownership, or provider internals.
- Adapter-private GitHub DTOs, runner, semantic mapping, and current GitHub
  metadata compatibility structs.
- New permissions: planning.collaboration and planning.publish.
- New events: planning.integration.configured,
  planning.collaboration.repaired, planning.publication.applied,
  planning.integration.adopted, planning.integration.reconciled, and
  planning.execution.status_updated.
- Dependency direction: planning <- planning/application <- planning/github;
  Brain module and both CLI hosts wire dependencies from outside.

## Risks / Open Questions

- Generic contracts may leak GitHub nouns. Guard exports/dependencies and review
  types before adapter implementation.
- Fake-client tests may omit undocumented script consumers. Pin exact JSON/file
  effects and normalized human output before migration.
- Remote partial failure can leave objects without mappings. Preserve managed
  source identity and recover on next preview; never auto-delete.
- Pagination, rate limits, API previews, and Project option-ID churn can create
  provider drift. Preserve bounded lookups and typed manual remediation.
- Brain/Plan release skew can split behavior. Land Brain first, publish a stable
  tag, then pin it in Plan without replace.
- Config version 2 changes validation. Test v1/default migration, disabled
  behavior, and rollback before native enablement.
- GitHub source mode remains supported through Phase 9 replacement and a
  separate reviewed deprecation decision. Phase 5 has no removal warning/date.
- No unresolved approval question remains.

## Rollout

1. Add Plan v0.1.29 conformance manifest and fake-provider transcripts without
   behavior changes.
2. Land provider-neutral DTOs/ports/export and dependency guards.
3. Add planning/github transport plus metadata fixtures behind disabled config.
4. Migrate collaboration assessment/repair and publication/adoption.
5. Migrate repository evidence, reconciliation, Project drift/status, native
   commands, permissions, events, audit, and health.
6. Publish merged Brain boundary as stable release.
7. Pin release in Plan, route mapped GitHub commands, preserve compatibility-only
   story/source behavior, and repeat full matrix.
8. Publish Plan and update transition/support docs before Phase 5 completion.

Each slice is independently revertible. Rollback disables native wiring or
restores a standalone wrapper to Plan-owned behavior; artifacts and mappings
need no backfill.

## Verification

- Run focused application port/service, GitHub adapter, config, permission,
  event/audit, CLI, metadata, partial-failure, recovery, and idempotency tests
  with injected fake transport.
- Replay manifest through Brain native and Plan wrapper paths against independent
  fixtures and exact fake-provider transcripts.
- Test revision conflict, ambiguous identity, partial apply, state write failure
  after remote success, Project drift, unsupported field options, unavailable or
  unauthenticated gh, provider errors, pagination limits, and unchanged reruns.
- Test disabled module/adapter with failing spy runner to prove zero preflight,
  network, GitHub metadata write, command execution, or event.
- Add guards proving planning, planning/application, and planning/local cannot
  import planning/github, Cobra, Brain internals, GitHub SDKs, Linear, or cloud.
- Test argument boundaries, cancellation/timeouts, stderr/redaction, repository
  paths with spaces, and Windows command/path behavior.
- Run go test ./..., go test -race ./..., go vet ./..., go build ./..., Windows
  build/test compilation, go mod verify, and Plan checks in both repositories.
- Publish Brain first; pin stable tag in Plan; repeat cross-repo verification;
  publish Plan; verify checksums and six platform archives.

## Execution Plan

Derive runtime slices only when implementation begins. Recommended order:
conformance baseline; provider-neutral contract; disabled adapter skeleton;
collaboration/publication; repository/Project reconciliation; Plan wrapper and
coordinated release.

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

- [Source Brainstorm](../brainstorms/github-adapter-transition.md)
- [GitHub Planning Transition](../../docs/planning/github-transition.md)
- [ADR 0007](../../docs/adr/0007-github-planning-is-transitional.md)
- [Planning Storage and Modes](../../docs/planning/storage-and-modes.md)
- [Planning Domain Model](../../docs/planning/domain-model.md)
- [Standalone Plan Feature Inventory](../../docs/planning/plan-feature-inventory.md)
- [Planning CLI Migration](../../docs/planning/cli-migration.md)
- [Phase 4 Plan CLI Compatibility](./plan-cli-compatibility.md)

## Notes

- Standalone local discuss promotion remains preview-only. Spec was seeded
  through legacy local promotion compatibility; generated epic was intentionally
  removed because Brain Planning uses spec-first model.
- Plan v0.1.29 at 898b2a4c4703 is frozen behavioral baseline, not target layout.
