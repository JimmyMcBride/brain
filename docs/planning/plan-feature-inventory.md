# Standalone Plan Feature Inventory

## Authority and Snapshot

Migration checklist based on `JimmyMcBride/plan` `develop` at commit `53ebd96`
(inspected 2026-07-27), including implementation and tests rather than README
claims alone.

Classifications:

- **PRESERVE**: retain behavior/domain intent
- **REDESIGN**: retain need through Brain-native contracts
- **TRANSITIONAL**: support during migration with planned exit
- **RETIRE**: remove after compatibility handling
- **UNDECIDED**: needs prototype or policy decision

## Feature and Command Matrix

| Feature / command | Current location | Current behavior and persistence | Dependencies | Class | Target owner | Risk / compatibility | Phase |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `init` | `cmd/init.go`, `internal/workspace` | Creates `.plan/` tree/templates/meta JSON | filesystem/templates | REDESIGN | Planning local adapter + Core module lifecycle | Must not create when disabled; preserve fixture compatibility | 3 |
| `adopt` | `cmd/adopt.go`, workspace | Registers existing repo and creates missing workspace surfaces | filesystem | PRESERVE | Planning local adapter | Non-destructive adoption semantics | 3 |
| `doctor` | `cmd/doctor.go`, workspace | Reports initialized/adoptable/migration health | filesystem/JSON | PRESERVE | Planning health provider | Old/Linear/future schemas need actionable diagnostics | 1,3,6 |
| `update` / archive legacy | `cmd/update.go`, workspace | Reconciles meta defaults; optional epic/story archive | filesystem/time | TRANSITIONAL | Planning migrator | Backup and rerun idempotency | 3,6 |
| `check` | `cmd/check.go`, `planning/check.go` | Project/artifact quality and execution-readiness findings | parsed Markdown | PRESERVE | Planning readiness | Preserve finding severity/JSON behavior | 2,3 |
| `status` | `cmd/status.go`, manager | Aggregates roadmap/spec/story planning status | Markdown/meta | PRESERVE | Planning query service | Legacy counts need mapping | 2,3 |
| `source show/set` | `cmd/source.go`, `source_mode.go`, workspace | Selects local/GitHub/hybrid/Linear ownership | `workspace.json`; integrations | REDESIGN | Planning storage/ownership policy | Script/JSON compatibility; Linear diagnostic | 3,5,6,9 |
| `roadmap show/edit` | `cmd/roadmap.go`, `roadmap.go` | Reads/replaces human-authored `ROADMAP.md` | filesystem/editor | PRESERVE | Planning domain/local adapter | Preserve Markdown and edit semantics | 2,3 |
| Brainstorm create/show | `cmd/brainstorm.go`, `manager.go` | Creates/shows slugged discovery Markdown | `.plan/brainstorms/*.md` | PRESERVE | Planning | Stable slug/frontmatter/content | 2,3 |
| Brainstorm idea | same | Appends idea/constraint/open question | brainstorm Markdown | PRESERVE | Planning | Section mapping and repeat behavior | 2,3 |
| Brainstorm refine | `refinement.go`, command tests | Interactive/resumable clarification clusters | brainstorm `## Refinement`; guided meta | PRESERVE | Planning | TTY/agent flow and partial saves | 2,3 |
| Brainstorm challenge | `challenge.go` | Captures rabbit holes/no-gos/assumptions/simpler alternative | brainstorm `## Challenge` | PRESERVE | Planning | Preserve additive non-destructive pass | 2,3 |
| Guided start/resume/switch/reopen/review | `guided_session.go`, `cmd/brainstorm.go` | Chain-scoped staged co-planning | `.meta/guided_sessions.json` + artifacts | PRESERVE | Planning workflow over Core services | Resume and downstream-needs-review behavior | 2,3 |
| Roadmap parking | guided session + brainstorm command | Parks future idea with unlock condition/source link | idea/roadmap artifact | PRESERVE | Planning | Stable source links | 2,3 |
| Guide packets | `guide_packet.go`, `cmd/guide.go` | Emits typed stage/question/artifact/action contract for agents | computed JSON + session meta | REDESIGN | Planning agent tools | Exported JSON likely agent-consumed; version it | 3,7 |
| Maturity assessment | `collaboration.go`, briefs, `cmd/discuss.go` | not-ready/repair/single/multi-spec decision with gaps | computed JSON; local or GitHub source | PRESERVE | Planning | Decouple GitHub source; preserve states | 2,3,5 |
| Source repair | collaboration + command | Adds canonical Specs split locally or to confirmed Discussion | Markdown or GitHub mutation | PRESERVE | Planning + collaboration adapter | Remote confirmation; idempotent source edit | 2,5 |
| Promotion draft | collaboration/reconcile | Preview create/update/reuse/unchanged issue/relationship actions | computed JSON | PRESERVE | Planning application | Stable identity, no title guessing | 2,3 |
| Promotion apply | collaboration/reconcile | Confirmed apply for GitHub/hybrid and Linear packet path | remote + meta JSON | REDESIGN | Planning publisher adapters | Mutation/confirmation/idempotency contract | 5,6 |
| Promotion confirmation policy | collaboration types/tests/skill | Preview non-mutating; apply requires `--confirm`; manual fallback restricted | JSON policy | PRESERVE | Planning + Core tool policy | Security-critical compatibility | 1,3,5 |
| Initiative | `initiative.go`, spec command | Lightweight grouping metadata; multi-spec promotion | spec Markdown; milestone mapping | PRESERVE | Planning | Remove GitHub identity from domain | 2 |
| Specs show/edit/status | `cmd/spec.go`, manager | Canonical Markdown; draft/approved/implementing/done | `.plan/specs/*.md` | PRESERVE | Planning | Canonical sections/status and editor compatibility | 2,3 |
| Spec analyze | `analyze.go` | Reports refinement gaps without rewriting canonical sections | computed/report content | PRESERVE | Planning readiness | Preserve non-mutating guarantee | 2,3 |
| Spec checklists | `checklist.go` | General/UI/API/data-migration profile findings | additive report/checklist | PRESERVE | Planning readiness extension | Profiles may become extension point | 2,3 |
| Spec initiative metadata | command + `initiative.go` | Set/clear initiative slug/title/summary | spec frontmatter/body | PRESERVE | Planning | Stable references | 2,3 |
| Spec execute | `spec_execute.go` | Suggested branch plus ordered ephemeral execution slices | computed plan; spec status | PRESERVE | Planning execution | One-slice discipline; no forced tiny files | 2,3 |
| Spec handoff | command + guided session | Moves guided chain into execution at checkpoint | guided-session JSON | PRESERVE | Planning workflow | Resume/idempotency | 2,3 |
| Execution queues | status, GitHub project status, spec execute | Ready/current/blocked work inferred from specs/stories/issues | local artifacts/GitHub | REDESIGN | Planning execution | Consolidate competing models | 2,5 |
| Runtime slices | `SpecExecutionSlice`, `story_slice.go` | Ephemeral spec slices and legacy persisted story candidates | computed or story Markdown | PRESERVE | Planning execution | Preserve small-slice behavior without mandatory files | 2 |
| Handoffs | epic/spec guided handoff | Stage transitions with generated artifacts | Markdown + session JSON | PRESERVE | Planning workflow | Legacy epic path transitional | 2,3 |
| Epic create/list/show/promote/shape/handoff | `cmd/epic.go`, shape/manager | Legacy hierarchy and seeded spec | `.plan/epics`/archive + specs | RETIRE | Compatibility importer | Read/migrate links/content; stop new use later | 4,6 |
| Story create/update/list/show | `cmd/story.go`, manager | Legacy execution items after approved spec | `.plan/stories`/archive or GitHub | RETIRE | Compatibility/external work mapping | Status/dependency/history preservation | 4,5,6 |
| Story slice | `story_slice.go` | Preview/confirm first-pass stories from approved spec | computed then story notes | TRANSITIONAL | Planning runtime slices | Preserve preview semantics while changing model | 2,4 |
| Story critique | `critique.go` | Interactive execution-readiness review | story Markdown | TRANSITIONAL | Planning readiness on slices | Migrate useful checks | 2,4 |
| Local source mode | `source_mode.go`, workspace | `.plan/` owns durable planning | filesystem | PRESERVE | Planning local adapter | Default/no hosted requirement | 3 |
| GitHub source mode | source/workspace/GitHub backend | GitHub can own promoted artifacts/execution | GitHub + local mappings | TRANSITIONAL | GitHub companion adapter | Do not remove before Cloud/adapter path | 5 |
| Hybrid source mode | source/workspace | Explicit split across local/integration layers | filesystem/GitHub | REDESIGN | Planning ownership/sync | Current semantics narrower than target hybrid | 5,9 |
| Linear source mode | source/workspace | Linear team/project/issue metadata and agent-mediated path | `linear.json`, MCP workflow | RETIRE | Read-only migrator/community future | Explicit unsupported/migration message | 6 |
| GitHub Discussions | `github_client.go`, collaboration | Read source/comments; repair/promotion | GitHub | TRANSITIONAL | Collaboration adapter | Preserve source URL/revision evidence | 5 |
| GitHub Issues | backend/reconcile/adopt | Initiative/spec/story records | GitHub Issues + `github.json` | TRANSITIONAL | Publication/external work adapter | Stable IDs and action plans | 5 |
| GitHub Projects | project workspace/status/drift | Workspace fields/items and execution status | GitHub Projects v2 + meta | TRANSITIONAL | Execution-status adapter | GraphQL IDs/field options/drift | 5 |
| GitHub milestones | collaboration/reconcile | Multi-spec grouping/release-like marker | GitHub + mappings | TRANSITIONAL | Publication adapter | Preserve number/title mapping | 5 |
| GitHub reconciliation | `github_reconcile.go`, project drift | Reconciles merged planning and optional visible readiness | Git/gh/GitHub/meta | PRESERVE semantics | GitHub adapter | Idempotency and partial failure | 5 |
| GitHub adoption | `cmd/github.go`, collaboration | Maps manually/externally created issues in draft order | `github.json` | TRANSITIONAL | GitHub adapter/migrator | Ambiguous ordering/identity | 5 |
| GitHub project execution status | `github_project_status.go` | Moves tracked issue todo/in-progress/in-review/done | Project item fields | TRANSITIONAL | Execution-status adapter | Preserve status mapping during overlap | 5 |
| GitHub client abstraction | `github_client.go` | Broad GitHub operations including discussions/issues/projects | `gh`/API implementation | REDESIGN | GitHub companion module | Split by capability; keep types out of domain | 5 |
| `.plan/.meta/workspace.json` | `WorkspaceMeta` | version, source mode, ownership, story backend | tool JSON | TRANSITIONAL | Planning local metadata | Versioned read/migrate | 3,9 |
| `.plan/.meta/migrations.json` | `MigrationState`, archive records | migration runs and archived paths/specs | tool JSON | PRESERVE/evolve | Planning migrator | Audit, backup, idempotency | 3 |
| `.plan/.meta/github.json` | `GitHubState` and records | repo/integration/planning/story/project mappings | tool JSON | TRANSITIONAL | GitHub adapter metadata | Preserve IDs and last reconciliation | 5 |
| `.plan/.meta/linear.json` | `LinearState` | workspace/team/project/issue identity | tool JSON | RETIRE | Read-only migration input | Never silently delete | 6 |
| `.plan/.meta/guided_sessions.json` | guided workspace file | active chains, stages, checkpoints, actions | tool JSON | PRESERVE/evolve | Planning workflow state | Schema/version and resume compatibility | 2,3 |
| Markdown templates | `internal/templates` | Project/roadmap/brainstorm/epic/spec/story canonical shapes | embedded templates | PRESERVE/evolve | Planning local adapter | Golden fixtures before edits | 2,3 |
| JSON command outputs | collaboration/guide/status/source/etc. structs | Agent/script machine contracts | stdout | REDESIGN/version | Planning API/CLI | Inventory fields and golden outputs before change | 3,4 |
| Workspace migrations | `internal/workspace` reconcile/archive | Normalizes defaults and archives legacy paths | filesystem/meta JSON | PRESERVE | Planning migrator | Backup, preview, unknown version, rerun | 3 |
| Planning tests | `cmd/*_test.go`, `internal/planning/*_test.go`, workspace tests/testdata | CLI, domain, GitHub fake-client, guided, eval, idempotency coverage | Go tests/fixtures | PRESERVE | Relevant target package | Port behavior, not package layout | all |
| Eval fixtures/rubrics | `internal/planning/evals.go`, `testdata/evals` | Scores planning output quality | JSON fixtures | PRESERVE | Planning quality tests | Avoid coupling runtime to benchmark format | 2 |
| Agent policies/skills | `AGENTS.md`, `skills/plan*` | Enforces specs, confirmation, source ownership, execution loop | agent instructions | REDESIGN | Brain + Planning tools/skills | Keep safety rules during CLI transition | 3,4 |
| Idempotency behavior | promotion reconcile/apply, workspace reconcile, GitHub tests | Explicit action plans; repeat apply/update safe | stable IDs/meta | PRESERVE | Core mutation policy + Planning/adapters | Must receive dedicated contract tests | 1,3,5 |

## Current Persistence Schema Summary

`WorkspaceMeta` contains workspace version, source-of-truth mode, ownership map,
story backend, and update timestamps. `GitHubState` contains repository identity,
integration state, planning records, project decision/workspace records, story
records, reconciliation timestamps, and version. `LinearState` contains
workspace/team identity plus planning project/issue mappings and promotion
metadata. `MigrationState` records schema/migration runs and archive manifests.
`GuidedSessionState` contains chain records, last-active chain, stage/checkpoint
state, downstream review state, and next actions.

Current workspace schema version is `3`; planning model is `spec_first_v1`
(`epic_spec_story_v1` is legacy).

| JSON contract | Current fields / nested records | Compatibility disposition |
| --- | --- | --- |
| `.plan/.meta/workspace.json` / `WorkspaceMeta` | `schema_version`, `planning_model`, `source_mode`, `story_backend`, `created_at`, `updated_at` | Golden fixture; translate ownership/config explicitly |
| `.plan/.meta/migrations.json` / `MigrationState` | schema, known migrations, last run/status/operation, created/updated/archived lists, history of operation/time/status/path changes | Preserve history and rerun safety |
| archive manifest | `archived_at`, `planning_model`, source path mappings, active spec records with legacy-epic source | Preserve for legacy recovery/import |
| `.plan/.meta/github.json` / `GitHubState` | repo URL/default branch, enabled/updated/reconciled times, maps for stories/planning/project decisions | Transitional adapter fixture |
| `GitHubPlanningRecord` | slug/kind/title, issue identity/state/readiness, ownership/entry/source modes, Discussion identity, parent/milestone, dependencies, update time | Preserve stable remote identity and links |
| `GitHubProjectDecisionRecord` | decision/reason, initiative/spec/milestone identity, Project owner/number/node/URL, field IDs, source/entry/Discussion, update time | Transitional; preserve decisions and GraphQL IDs |
| `GitHubStoryRecord` | domain content/status/dependencies/readiness critique, issue/PR/doc refs, ready/blocked/visible marker, update time | Map legacy story evidence without loss |
| `.plan/.meta/linear.json` / `LinearState` | workspace/team identity, enabled/updated/reconciled times, source and promotion target | Read-only diagnostic/export; retire |
| `.plan/.meta/guided_sessions.json` / `GuidedSessionState` | schema, last active/updated, keyed records containing chain/artifact refs, stage/cluster/checkpoint, stage statuses, summary/next action/times | Preserve/evolve with versioned workflow migration |
| maturity assessment stdout | schema/kind/time, source, ownership, decision state/confidence/reason/strengths/gaps/path/titles/dependencies/repair/next command | Versioned Planning API/CLI contract |
| promotion draft stdout | assessment plus issue drafts/actions/readiness, relationships, milestone/project plans, agent mutation policy, fallback and confirmation | Preserve safety/idempotency semantics; version schema |
| promotion apply/adopt stdout | draft plus created/adopted GitHub objects, project/milestone/parent data, fallback/next command | Transitional GitHub adapter output |
| source repair stdout | schema/kind/time/source/spec split, updated path/URL, next commands | Preserve local behavior; adapter remote mutation |
| guide packet stdout | builder/workspace/ownership/session/artifact/mode/sources, optional collaboration, question/artifact strategy, policies, drafts/actions/rendered prompt | Version and compatibility-test agent consumers |
| doctor/status/check/execute stdout | workspace health; aggregate status/findings; execution spec/branch/slices | Capture exact golden forms before Phase 3 |

Before Phase 3, capture exact JSON golden fixtures from current supported Plan
releases. This architecture intentionally does not freeze field-level target
schemas.

## Current Idempotency and Migration Behavior

- workspace update creates missing tool-managed files, normalizes supported old
  values, records migration runs, and rejects newer guided-session schema versions
- legacy archive records source/destination mappings and active spec provenance
- promotion reconciliation computes `create`, `update`, `reuse`, or `unchanged`
  before mutation and uses stored issue/milestone/project identity
- repeat confirmed GitHub promotion is covered by collaboration/reconcile tests
- source repair and adoption keep explicit source identity and ordered issue
  mapping
- GitHub reconcile records last-reconciled state and separates optional visible
  readiness mutation

Target migrations must preserve these properties and add explicit backup/preview
coverage wherever current normalization could overwrite malformed state.

## Package Classification

| Current package | Classification | Reason |
| --- | --- | --- |
| `internal/planning/manager.go` | REDESIGN | Domain, Markdown persistence, and application orchestration are mixed |
| refinement/challenge/analyze/check/checklist/initiative/roadmap | PRESERVE | Mostly domain behavior; isolate I/O |
| collaboration/briefs/reconcile | PRESERVE + REDESIGN | Valuable maturity/action semantics; source/backend coupling |
| `github_*` | TRANSITIONAL | Retain through adapter migration |
| `source_mode.go` | REDESIGN | Storage ownership replaces permanent provider-as-source model |
| guided session/guide packet | PRESERVE + REDESIGN | Keep behavior; integrate Core sessions/tools |
| story/epic code | RETIRE/TRANSITIONAL | Compatibility only |
| `internal/workspace` | REDESIGN | Split local schema/migration from integration adapters |
| `cmd` | TRANSITIONAL | Map to shared services and compatibility wrapper |
| `internal/skills` | REDESIGN | Fold into Brain module/tool discovery |

## Inventory Exit Rule

An item leaves this checklist only when its migration phase documents: target
contract, compatibility fixtures, migration behavior, tests, and deprecation or
preservation outcome.
