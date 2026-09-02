---
created_at: "2026-09-02T14:17:49Z"
epic: github-adapter-transition
project: brain
slug: github-adapter-transition
spec: github-adapter-transition
status: promoted
title: GitHub Adapter Transition
type: brainstorm
updated_at: "2026-09-02T14:22:24Z"
---

# Brainstorm: GitHub Adapter Transition

Started: 2026-09-02T14:17:49Z

## Focus Question

How should Phase 5 preserve standalone Plan GitHub collaboration, publication, repository evidence, work-item, milestone, Project, and reconciliation behavior behind provider-neutral Planning ports without making GitHub canonical or expanding into cloud, Linear, or hybrid synchronization?
## Desired Outcome

Brain Planning and standalone Plan share one provider-neutral application
workflow for retained GitHub collaboration and repository integration. GitHub
transport, API values, node IDs, labels, milestones, Projects, and compatibility
metadata live in one optional adapter. Existing GitHub and hybrid workspaces keep
working without artifact migration, local Planning stays network-free, and
GitHub remains transitional rather than becoming Planning's permanent backend.

## Vision

An agent can start from a GitHub Discussion, assess and repair its planning
content, preview an explicit promotion plan, publish or adopt initiative/spec
issues, preserve relationships and milestones, connect an optional Project,
track execution status, and reconcile repository evidence after merge. The same
application decisions drive `brain plan` and compatibility `plan` commands.
Provider-neutral Planning code sees collaboration sources, publication actions,
repository evidence, external references, and execution status—not GitHub API
objects. Disabling the adapter removes network behavior without weakening local
Planning.

## Supporting Material

- [GitHub Planning Transition](../../docs/planning/github-transition.md)
- [ADR 0007: GitHub Planning Support Is Transitional](../../docs/adr/0007-github-planning-is-transitional.md)
- [Planning Storage and Modes](../../docs/planning/storage-and-modes.md)
- [Planning Domain Model](../../docs/planning/domain-model.md)
- [Standalone Plan Feature Inventory](../../docs/planning/plan-feature-inventory.md)
- [Migration from Standalone Plan](../../docs/planning/migration-from-plan.md)
- [Planning CLI Migration](../../docs/planning/cli-migration.md)
- [Phase 4 Plan CLI Compatibility](../specs/plan-cli-compatibility.md)
- Standalone Plan compatibility baseline: `v0.1.29` / `898b2a4c4703`

## Constraints

- Brain Core and local Planning remain useful with no GitHub account, `gh`
  executable, network access, adapter configuration, or GitHub metadata.
- Planning domain, `planning/application`, and `planning/local` expose no GitHub
  imports or API-shaped types. Generic ports model only proven Planning needs.
- GitHub is an optional compiled official adapter. Phase 5 does not design the
  future external community-module protocol.
- Existing `local`, `github`, and `hybrid` values remain readable. Phase 5
  preserves current split ownership but does not design Phase 9 hybrid sync.
- `.plan/.meta/github.json` remains readable in place and retains its current
  observable shape during the compatibility overlap; no workspace backfill is
  required.
- Specs remain canonical. Initiative issues, milestones, Project items, and
  legacy stories are external tracking or compatibility records, not a new
  canonical hierarchy.
- Mutations stay preview-first, explicitly confirmed, permissioned, audited,
  action-classified, and idempotent. Ambiguous identity fails closed.
- GitHub credentials remain owned by `gh`; Brain and Plan persist no token.
- Linear, cloud storage, generalized hybrid synchronization, Brain knowledge
  proposals, legacy story creation, and source-mode retirement are excluded.

## Open Questions

- None. Refinement resolves package ownership, port granularity, command
  mapping, adapter enablement, metadata compatibility, failure recovery, and
  transition timing.

## Ideas

- Extract capability-sized provider-neutral ports from proven GitHub behaviors; do not mirror the broad GitHub client interface.
- Preserve preview, confirmation, stable identity, explicit actions, partial-failure safety, and idempotent reruns through conformance fixtures.
- Keep standalone Plan as the GitHub compatibility host while Brain owns public Planning contracts and an optional compiled GitHub adapter.
## Raw Notes

- Current standalone surface: remote `discuss assess/repair/promote`; `github
  enable/adopt/reconcile/project status`; GitHub-backed story compatibility;
  source show/set; GitHub-aware check and guide packets.
- Current broad client mixes repository context, Discussions, issues, labels,
  milestones, issue relationships, Projects, items, fields, and status writes.
  That interface is evidence for several capability ports, not a target API.
- Current metadata stores repository identity, planning records, Project
  decisions, story records, source provenance, remote identity, relationships,
  and reconciliation timestamps in `.plan/.meta/github.json`.
- Current safe drift repair creates missing Project fields/items and updates
  supported values, but does not mutate existing single-select options because
  GitHub regenerates option IDs. Preserve that manual-remediation boundary.

## Refinement

### Problem

Standalone Plan's GitHub behavior works, but its application layer contains
GitHub-specific Discussions, issues, milestone, Project, GraphQL, repository,
and metadata types. Moving that code directly into Brain would make a
transitional provider part of Planning's permanent contracts, couple local
Planning to hosted infrastructure, and preserve legacy story hierarchy by
accident. Leaving it entirely in Plan would prevent `brain plan` from becoming
the primary Planning entrypoint for collaborative and repository-backed work.

### User / Value

Agents and maintainers using GitHub keep their working Discussion-to-spec,
publication, adoption, Project, and reconciliation workflows while gaining one
Brain-owned application contract and native permission/audit behavior. Local
users pay no GitHub cost. Maintainers get capability-sized seams that can later
support Brain Cloud without pretending every provider has GitHub's object model.

### Appetite

One bounded Phase 5 spec, delivered incrementally across Brain then standalone
Plan. Freeze current behavior first; add only ports proven by retained commands;
implement one importable official GitHub adapter; wire native commands; migrate
standalone wrappers; publish coordinated stable releases. Do not add a universal
provider SDK, change `.plan/` artifacts, retire source modes, or redesign hybrid
sync.

### Remaining Open Questions

- None.

### Candidate Approaches

- Recommended — capability ports plus official adapter: add provider-neutral
  collaboration-source, publication, repository-evidence, external-reference,
  and execution-status contracts to `planning/application`; implement GitHub
  transport and compatibility metadata in importable `planning/github`; wire it
  only when explicitly enabled in Brain's Planning module. Both CLIs retain
  presentation adapters.
- Plan-owned adapter only: define Brain ports but leave all GitHub implementation
  in standalone Plan. Lowest migration risk, but `brain plan` cannot own the
  retained workflow and the two hosts continue to drift.
- Broad client extraction: move the current `GitHubClient` and structs into
  Brain mostly unchanged. Fastest mechanical port, but turns GitHub API shape
  into public Planning architecture and blocks Cloud-neutral evolution.
- Universal provider/plugin protocol: design a generic out-of-process adapter
  SDK now. Future-looking but premature; Phase 11 owns community protocol work.

### Decision Snapshot

Use capability-sized provider-neutral ports and one official importable GitHub
adapter. `planning/application` owns orchestration DTOs, deterministic action
plans, stable external references, and ports. `planning/github` owns `gh`
execution, GitHub API values, managed-body/label conventions, remote lookup,
Project field details, and `.plan/.meta/github.json` compatibility. Brain's
official Planning module supplies configuration, permissions, audit/events, and
CLI presentation; standalone Plan pins a stable Brain release and supplies its
existing flags/output wrapper.

Native remote collaboration maps to `brain plan brainstorm
assess|repair|promote --discussion`. Provider administration remains explicit
under `brain plan github enable|adopt|reconcile|project status`. Phase 5 adds no
native `source set` and no native story creation. Existing GitHub-backed story
commands remain Plan-owned compatibility surfaces; they may consume shared
adapter primitives, but do not enlarge Brain's canonical model.

Planning module configuration version 2 gains only an explicit GitHub-adapter
enablement flag. Adapter enablement is separate from artifact ownership:
existing `github`/`hybrid` workspace values are compatibility inputs, and a
local workspace may still use a Discussion as a collaboration source. Disabled
adapter means no `gh` preflight, network call, GitHub-state mutation, or GitHub
command execution.

Use existing `planning.read` for remote reads, add
`planning.collaboration` for Discussion repair, add `planning.publish` for
publication/adoption/reconciliation mutations, and retain `planning.execute`
for Project execution-status changes. Every real mutation emits a typed Planning
event and Brain audit record; previews and unchanged reruns emit none.

Partial remote failure does not attempt destructive rollback. Results report
completed actions and the failing action; reruns re-read remote identity plus
adapter metadata and classify each action as `create`, `update`, `reuse`, or
`unchanged`. Manual `gh` fallback remains forbidden unless the result explicitly
sets `manual_fallback_allowed=true`.

Keep GitHub source mode supported through the future Phase 9 ownership/sync
replacement and a separate reviewed deprecation decision. Phase 5 publishes no
removal date and adds no deprecation warning.

## Challenge

### Rabbit Holes

- Reproducing every GitHub API operation as a supposedly generic Planning port.
- Turning issue, milestone, Project, label, or node-ID concepts into domain
  aggregates because the current provider exposes them.
- Migrating legacy story creation into native Brain Planning.
- Designing cloud adapters, hybrid conflict resolution, community transport, or
  a provider marketplace before the GitHub seam is proven.
- Rewriting `.plan/.meta/github.json`, moving `.plan/`, or introducing a second
  integration ledger during an open compatibility window.
- Adding direct OAuth/token storage or a GitHub SDK migration when injected `gh`
  execution already provides the proven behavior.
- Using live GitHub resources in required CI and creating flaky, destructive
  integration tests.

### No-Gos

- No GitHub, GraphQL, issue, milestone, Project, label, or Discussion types in
  Planning domain, `planning/application`, or `planning/local` public contracts.
- No GitHub import or network access from Brain Core or local-only workflows.
- No Linear type, state reader, adapter, command, permission, event, or fallback.
- No automatic remote rollback, title-only identity matching, silent mutation,
  or unconfirmed publication.
- No automatic editing of existing Project single-select options.
- No new canonical epic/story hierarchy and no persisted runtime slices.
- No source-mode removal, cloud/hybrid sync protocol, or support-window date.
- No checked-in local Go replacement between Brain and standalone Plan.

### Assumptions

- Plan `v0.1.29` tests and fake clients capture the retained GitHub baseline
  strongly enough to build a versioned conformance manifest.
- Existing remote objects expose stable number/node/URL identity and managed
  markers sufficient for safe reconciliation after partial failure.
- The current GitHub metadata shape can remain unchanged while ownership moves
  into the adapter.
- `gh` remains the initial authenticated transport and is available only when a
  user enables the adapter.
- Brain's compiled official-module model can include an optional adapter package
  without making it a Core dependency or a community protocol precedent.
- Capability ports proven by GitHub can later support Cloud with provider-native
  adapters, without promising identical features.

### Likely Overengineering

A universal forge API, generic GraphQL layer, provider registry, remote
transaction engine, durable operation journal, automatic rollback coordinator,
or community plugin protocol would exceed Phase 5. So would modeling every
Project custom field or issue relationship generically. Preserve only the
current Planning semantics and keep provider mechanics private.

### Simpler Alternative

Freeze Plan `v0.1.29` behavior; extract the smallest provider-neutral requests,
results, and ports needed by remote assess/repair/promote, adopt, reconcile, and
Project status; place the existing `gh` mechanics behind `planning/github`;
wire explicit Brain configuration and permissions; then switch the same
standalone commands to the shared service. Leave source mutation and legacy
story creation in standalone compatibility code.

## Promotion map

### Spec 1 — GitHub Adapter Transition

Move retained GitHub collaboration, publication, repository evidence,
external-work, Project-status, and reconciliation behavior behind
provider-neutral Planning application ports and one optional official GitHub
adapter, while preserving standalone Plan compatibility and current workspace
artifacts.

Scope:

Pin Plan `v0.1.29` as the compatibility baseline and capture versioned remote
command/fake-provider contracts. Add capability-sized provider-neutral DTOs,
ports, action plans, results, errors, and services in `planning/application`.
Add importable `planning/github` transport and compatibility-state adapters.
Wire explicit adapter configuration, permissions, events/audit, health, and
native remote collaboration/GitHub command families through Brain's official
Planning module. Migrate mapped standalone Plan commands to the stable Brain
release while preserving presentation, fallbacks, and GitHub/hybrid ownership.
Keep legacy story creation and source mutation outside native Brain Planning.

Acceptance criteria:

- A versioned conformance manifest pins Plan `v0.1.29` and covers remote
  `discuss assess/repair/promote`, `github enable/adopt/reconcile/project status`,
  GitHub-aware check/guide behavior, repository/PR evidence, metadata effects,
  fake-provider calls, exit/stdout/stderr, confirmation, partial failure, and an
  identical rerun.
- `planning`, `planning/application`, and `planning/local` contain no GitHub
  imports or API-shaped exported types. Provider-neutral contracts model only
  collaboration content/revision, deterministic publication actions, repository
  evidence, stable external references/relationships, and execution status.
- `planning/github` owns `gh` execution, GitHub API values, managed labels/body
  markers, milestone/relationship/Project mechanics, and exact read/write
  compatibility for existing `.plan/.meta/github.json` fixtures.
- Brain Planning configuration version 2 enables GitHub explicitly and stores no
  credential. Disabled-module and disabled-adapter tests prove zero GitHub
  preflight, network access, GitHub metadata mutation, or command execution.
- Native remote collaboration is available through `brain plan brainstorm
  assess|repair|promote --discussion`; provider administration is available
  through `brain plan github enable|adopt|reconcile|project status`. No native
  `source set` or story-creation hierarchy is added.
- Remote reads require `planning.read`; Discussion repair requires
  `planning.collaboration`; publication/adoption/reconciliation requires
  `planning.publish`; Project status requires `planning.execute`. Confirmed real
  mutations emit typed events/audit exactly once; previews, denials, failures
  before mutation, and unchanged reruns emit none.
- Promotion/adoption/reconciliation preserve source links, stable slugs, remote
  number/node/URL identity, labels, milestone and parent/blocked-by relationships,
  Project decisions/fields/items, repository PR/branch/commit evidence, and
  current safe-drift/manual-remediation boundaries.
- Ambiguous identity fails closed. Partial failures return completed/failing
  actions and explicit fallback policy; reruns reconcile remote state and return
  `update`, `reuse`, or `unchanged` without duplicate objects or destructive
  rollback.
- Existing local, GitHub, and hybrid schema-v3 workspaces open in place with no
  artifact or metadata migration. Local workflows remain usable with no `gh`,
  network, adapter configuration, or GitHub state.
- Standalone Plan pins a published Brain version without `replace`, routes the
  mapped GitHub families through shared services/adapter, keeps JSON and script
  contracts, and retains GitHub-backed story/source commands only as documented
  compatibility surfaces.
- Docs state GitHub source mode remains supported through Phase 9 replacement
  and a separate deprecation decision; Phase 5 publishes no removal date.

Verification:

- Run focused application port/service, GitHub adapter, module config,
  permission/event/audit, CLI, metadata-fixture, partial-failure, and idempotency
  tests with injected fake transport; required CI creates no live GitHub object.
- Replay the conformance manifest against copied independent fixtures and exact
  fake-provider transcripts for Brain native and standalone wrapper paths.
- Run dependency/export guards proving provider-neutral packages cannot import
  `planning/github`, Cobra, Brain internals, GitHub SDKs, Linear, or cloud code.
- Run argument-boundary, timeout/cancellation, stderr/redaction, unavailable
  `gh`, unauthenticated, permission-denied, rate/error, and Windows-path tests.
- Run `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`,
  Windows build/test compilation, `go mod verify`, and project Plan checks in
  both repositories.
- Publish Brain first, pin that stable tag in Plan, repeat the cross-repository
  matrix, then publish the corresponding Plan release before marking Phase 5
  complete.

Dependencies: Phase 4 complete; Brain `v0.1.21`; Plan `v0.1.29` compatibility baseline.

Readiness: ready.
