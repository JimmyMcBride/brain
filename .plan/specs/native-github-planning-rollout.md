---
created_at: "2026-10-26T14:28:47Z"
initiative: github-adapter-transition
project: brain
slug: native-github-planning-rollout
status: draft
title: Native GitHub Planning Rollout
type: spec
updated_at: "2026-10-26T14:32:45Z"
---

# Native GitHub Planning Rollout

Created: 2026-10-26T14:28:47Z

## Why

Shared application services and the GitHub adapter are not usable as Brain
Planning until module configuration, host dependencies, commands, permissions,
events, audit, and health are wired together.

## Problem

The official Planning module remains config version 1, accepts no GitHub
configuration, constructs only the local adapter, and exposes no native GitHub
commands. Existing provider capabilities can be tested as packages but users
cannot enable, inspect, or run them through `brain plan`.

## Goals

- Add explicit, independently disableable GitHub adapter configuration.
- Wire completed collaboration, publication, adoption, reconciliation,
  repository-evidence, mapping, and execution capabilities into Brain Planning.
- Expose the approved native remote collaboration and GitHub administration
  command families.
- Enforce permissions, confirmation, audit/events, health, and output contracts.
- Preserve local Planning behavior when GitHub is absent, disabled, unhealthy,
  or unauthenticated.

## Non-Goals

- Adding native source-mode mutation.
- Adding native legacy story creation.
- Persisting GitHub credentials.
- Changing `.plan/` artifact ownership.
- Releasing standalone Plan; coordinated release follows separately.
- Adding live GitHub calls to required CI.

## Constraints

- Module config version 2 contains only reviewed adapter enablement settings;
  credentials remain owned by `gh`.
- Config v1/default migration and rollback are deterministic.
- Disabled adapter means zero preflight, command execution, provider call,
  GitHub metadata write, or GitHub event.
- Remote reads require `planning.read`; source repair requires
  `planning.collaboration`; publication/adoption/reconciliation/config changes
  require `planning.publish`; execution status requires `planning.execute`.
- Preview, denial, pre-mutation failure, and unchanged rerun emit no mutation
  event.
- GitHub health may degrade without making local Planning unhealthy.

## Solution Shape

### Module configuration and dependencies

Upgrade official Planning module configuration to version 2 with an explicit
GitHub-enabled value. Validate exact known keys, migrate the current empty v1
configuration safely, construct the optional GitHub adapter only through host
wiring, and expose independent health diagnostics.

### Native commands

Add remote collaboration forms:

```text
brain plan brainstorm assess --discussion <number-or-url>
brain plan brainstorm repair --discussion <number-or-url> --spec <title> --confirm
brain plan brainstorm promote --discussion <number-or-url> [--apply --confirm]
```

Add provider administration:

```text
brain plan github enable|disable
brain plan github adopt
brain plan github reconcile
brain plan github project status
```

Preserve preview-first output, JSON contracts, explicit confirmation, typed
errors, and no manual `gh` fallback unless the result explicitly allows it.

### Host policy

Register `planning.collaboration` and `planning.publish`, all Phase 5 integration
events, audit payloads, and adapter health. Map command input/output only in the
host; Planning application and GitHub adapter remain free of Cobra and Brain
private runtime dependencies.

### Compatibility surfaces

Extend native `check` and `guide` behavior for enabled GitHub workspaces using
shared contracts. Keep source mutation and GitHub-backed legacy story creation
in standalone Plan as documented compatibility-only behavior.

## Acceptance Criteria

- Config v1/default projects migrate to valid config v2 without enabling
  GitHub accidentally.
- Enable preflights `gh`, initializes compatible adapter state after
  confirmation, and is idempotent.
- Disable preserves artifacts/mappings and prevents all GitHub work.
- Every approved native command reaches shared services with correct
  permissions, confirmation, audit, event, and output behavior.
- Adapter/provider failure degrades GitHub health while local commands remain
  healthy.
- Machine output contains no prompts, warnings, secrets, or provider-only DTOs.
- Disabled-mode spy tests observe zero process, network, metadata, or event work.
- Dependency/export guards remain green.

## Data / Interfaces

- Official Planning module configuration advances from version 1 to version 2
  with one explicit GitHub enablement value.
- Application DTOs and adapter ports remain host-neutral; Cobra and Brain
  runtime types stay in host wiring.
- Registered permission/event names follow the parent Phase 5 contract.

## Risks / Open Questions

- Config-version mismatch can make Planning unavailable before users can
  migrate. Default/v1 migration and rollback need fixture coverage.
- Provider health must not contaminate local Planning health or trigger eager
  `gh` work during unrelated commands.
- Native and standalone output can drift even when behavior is shared; final
  conformance remains a separate release gate.
- No unresolved command-family decision remains; source mutation and legacy
  story creation stay outside Brain.

## Rollout

1. Add config v2 migration and disabled-mode tests.
2. Wire adapter dependencies and health without exposing commands.
3. Add native read/preview commands, then confirmed mutations.
4. Replay native manifest families and keep release behind final conformance.
5. Roll back by disabling GitHub configuration/commands; local Planning and
   durable GitHub mappings remain intact.

## Verification

- Config migration, validation, enable/disable, rollback, and health tests.
- CLI golden/JSON tests for every native manifest family.
- Permission, confirmation, audit/event, no-op, partial-failure, and redaction
  coverage.
- Independent local-only tests with missing `gh` and failing spy runner.
- Full test, race, vet, build, module, Windows, and Plan check matrix.

## Dependencies

- [GitHub Coordination Mapping Persistence](./github-coordination-mapping-persistence.md)
- [GitHub Repository Reconciliation](./github-repository-reconciliation.md)
- [GitHub Execution Workspace](./github-execution-workspace.md)
- Parent contract: [GitHub Adapter Transition](./github-adapter-transition.md)

## Follow-Up

[GitHub Integration Conformance and Release](./github-integration-conformance-release.md)
coordinates cross-host proof and publication.

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
