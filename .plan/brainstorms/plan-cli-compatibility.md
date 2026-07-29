---
created_at: "2026-07-29T08:36:48Z"
epic: plan-cli-compatibility
project: brain
slug: plan-cli-compatibility
spec: plan-cli-compatibility
status: promoted
title: Plan CLI Compatibility
type: brainstorm
updated_at: "2026-07-29T08:41:16Z"
---

# Brainstorm: Plan CLI Compatibility

Started: 2026-07-29T08:36:48Z

## Focus Question

How should Phase 4 make standalone plan and native brain plan share observable local behavior without pulling GitHub, Linear, cloud, or legacy creation into scope?
## Desired Outcome

`brain plan` provides a coherent local spec-first workflow while existing `plan`
users and automation retain compatible behavior during a documented overlap.
Both command families follow one explicit observable-behavior contract, and
every standalone command has a reviewed preserve, redesign, transition, defer,
or retire disposition.

## Vision

Planning feels native inside Brain without forcing users to abandon working
`.plan/` repositories or scripts in one jump. The standalone CLI remains a safe
compatibility entrypoint while local behavior converges on Brain-owned Planning
contracts. Compatibility is measurable: the same fixture and intent produce
equivalent results, diagnostics, machine output, filesystem effects, and
idempotent reruns. Later GitHub, Linear-cleanup, knowledge-loop, cloud, and hybrid
phases remain visibly separate.

## Supporting Material

- [Planning CLI Migration](../../docs/planning/cli-migration.md)
- [Standalone Plan Feature Inventory](../../docs/planning/plan-feature-inventory.md)
- [Migration from Standalone Plan](../../docs/planning/migration-from-plan.md)
- [Planning Module Implementation Roadmap](../../docs/planning/implementation-roadmap.md)
- [Planning Domain Extraction](../specs/planning-domain-extraction.md)
- [Local Planning Module](../specs/local-planning-module.md)
- Standalone Plan reference: `JimmyMcBride/plan` `develop` at `53ebd96`

## Constraints

- Brain Core remains useful with Planning disabled.
- Supported migration sources remain local, github, and hybrid only; no Linear types, files, commands, permissions, events, or adapter enter Brain.
- Specs remain canonical and execution slices are derived only when a spec workflow starts.
- Standalone plan must remain usable during the supported overlap.
- Warnings go to stderr and never corrupt JSON stdout.
- Compatibility covers observable behavior, not identical package layout or help prose byte-for-byte unless scripts consume it.
- GitHub behavior, cloud/hybrid sync, and legacy epic/story creation stay outside Phase 4 implementation.

## Open Questions

- Where should the shared application and compatibility implementation live while brain and plan remain separate Go modules?
- Which local command families are required for a coherent Phase 4 spec-first workflow, versus mapped but deferred?
- Should standalone plan delegate mapped commands to Brain, consume an importable shared package, or remain legacy-backed behind a conformance harness during overlap?
- When should migration warnings begin, and which invocations must remain warning-free for machine compatibility?
- Which documented JSON payloads and exit codes are hard compatibility contracts versus intentionally versioned redesigns?
- How will cross-repository compatibility tests pin Brain and Plan revisions reproducibly in CI?
## Ideas

- Freeze a full command-disposition matrix, but implement compatibility only for the local spec-first workflow owned by Phase 4.
- Use golden compatibility tests for flags, stdout, stderr, JSON, exit codes, filesystem effects, confirmation, and idempotency across both command families.
- Keep standalone plan functional during overlap, send migration warnings only to stderr, and defer removal timing until native coverage and workspace migration are proven.
- Resolve the two-repository shared-implementation boundary before approving the spec; do not introduce a third product or copy behavior between repositories.
## Raw Notes

## Refinement

### Problem

Standalone Plan remains the compatibility source while Brain now owns the first native local Planning workflow. Without an explicit compatibility contract and shared implementation boundary, the two CLIs will drift, scripts may break, and Phase 4 could accidentally absorb later GitHub, Linear-cleanup, cloud, or legacy-creation work.

### User / Value

Existing Plan users and automation keep working during migration, while Brain users gain one coherent local spec-first workflow under brain plan. Maintainers get one behavior contract, measurable compatibility, and a safe path to make Brain Planning primary without forcing an immediate standalone Plan retirement.

### Appetite

One bounded Phase 4 spec that establishes the compatibility architecture, maps the full standalone command inventory, and delivers a coherent local spec-first workflow through shared behavior. Implementation may derive several runtime slices, but the spec must not absorb Phase 5 integrations, Phase 6 Linear cleanup, or Phase 7 knowledge-loop work.

### Remaining Open Questions

- Where should the shared application and compatibility implementation live while brain and plan remain separate Go modules?
- Which local command families are required for a coherent Phase 4 spec-first workflow, versus mapped but deferred?
- Should standalone plan delegate mapped commands to Brain, consume an importable shared package, or remain legacy-backed behind a conformance harness during overlap?
- When should migration warnings begin, and which invocations must remain warning-free for machine compatibility?
- Which documented JSON payloads and exit codes are hard compatibility contracts versus intentionally versioned redesigns?
- How will cross-repository compatibility tests pin Brain and Plan revisions reproducibly in CI?

### Candidate Approaches

- Brain-owned importable package: normalize Brain's Go module path, expose only storage-neutral Planning/application compatibility packages, and make both command trees thin adapters over them.
- Conformance-first overlap: leave each repository's implementation in place temporarily, define golden black-box contracts, and migrate one command family at a time until Plan can become a wrapper; lowest initial coupling but allows short-lived duplication.
- Subprocess wrapper: standalone plan dispatches mapped local commands to brain plan and retains legacy handling for unmapped commands; simple ownership but adds an installation/runtime dependency and weakens standalone behavior.
- Nested shared module inside the Brain repository: both binaries consume a small versioned Planning module without creating a third repository or product; clean dependency direction but adds multi-module release complexity.

### Decision Snapshot

Use a conformance-first rollout toward Brain-owned importable Planning packages. First freeze black-box behavior for the bounded local spec-first command set. Then normalize Brain's module path and expose the smallest public domain/application/local compatibility boundary needed by both command trees. Migrate command families incrementally; standalone plan retains its existing implementation only for explicitly deferred commands. Reject subprocess delegation and a nested Go module because they add runtime or release complexity without improving the long-term ownership boundary.

## Challenge

### Rabbit Holes

- Treating Phase 4 as byte-for-byte parity for the entire standalone CLI.
- Moving GitHub collaboration and reconciliation forward from Phase 5.
- Importing standalone Plan packages into Brain and accidentally carrying Linear or provider-specific types across the boundary.
- Designing a generic plugin protocol, RPC layer, or third shared product for two local command trees.
- Rewriting the .plan schema while open-in-place compatibility already works.
- Implementing permanent epic/story creation instead of bounded read/import or deferred compatibility.
- Changing every Brain import path and public API without isolating the mechanical module-path work from compatibility behavior.

### No-Gos

- No Linear schema, adapter, state reader, command, permission, event, or compatibility package in Brain.
- No GitHub API, Discussion, issue, milestone, Project, or reconciliation implementation.
- No cloud or hybrid synchronization.
- No subprocess dependency from standalone plan to an installed brain binary.
- No nested Go module or third repository solely to share Phase 4 code.
- No warning on stdout and no warning that invalidates machine-readable JSON.
- No mandatory persisted slice files; slices remain runtime execution records.
- No standalone Plan retirement or published end-of-support date in this phase.

### Assumptions

- Brain can adopt a canonical GitHub Go module path without changing CLI behavior.
- A narrow public Planning package can remain independent of Cobra, Brain private storage, GitHub, Linear, and cloud dependencies.
- Standalone Plan can pin a Brain module version and remain independently buildable without the brain binary installed.
- The schema-v3 fixtures and current Plan command tests are sufficient to establish a compatibility baseline.
- A coherent local spec-first workflow can be bounded before GitHub adapter work.
- Warnings on stderr are acceptable only after the mapped command is proven compatible.

### Likely Overengineering

A broad exported SDK, generic command descriptor framework, universal compatibility engine, or cross-repository orchestration service would exceed the need. The public surface should be only typed Planning services and local adapters required by mapped commands; the compatibility harness can be test-only and table-driven.

### Simpler Alternative

Freeze the observable contract first, then expose only the existing Phase 3 local services as an importable Brain package and prove standalone plan can consume them for the already-overlapping status, brainstorm list/show/start, and spec list/show commands. Expand to the remaining local spec-first families only when each family has golden compatibility coverage; leave all other commands legacy-backed and explicitly mapped to later phases.
