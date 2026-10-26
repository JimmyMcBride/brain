---
created_at: "2026-10-26T14:28:47Z"
initiative: github-adapter-transition
project: brain
slug: github-repository-reconciliation
status: draft
title: GitHub Repository Reconciliation
type: spec
updated_at: "2026-10-26T14:32:45Z"
---

# GitHub Repository Reconciliation

Created: 2026-10-26T14:28:47Z

## Why

Teams need GitHub planning records to stay aligned after reviewed planning work
merges. Brain must know which repository change is authoritative before it can
propose a safe repair.

## Problem

Brain cannot yet determine whether GitHub planning records agree with the
repository change that was reviewed or merged. Users must return to standalone
Plan to repair issue content, readiness, relationships, and reconciliation
evidence, so Brain cannot complete the GitHub-backed workflow itself.

## Goals

- Identify the exact reviewed repository change being reconciled.
- Produce a deterministic preview from canonical Planning intent and observed
  remote state.
- Apply only reviewed safe repairs.
- Send unsupported or ambiguous differences to manual remediation.
- Record successful reconciliation only after all required work succeeds.
- Leave an identical rerun unchanged.

## Non-Goals

- Managing Project execution tracking.
- Creating, merging, or closing repository changes.
- General repository synchronization or Phase 9 hybrid ownership.
- Exposing the final end-user command surface.

## Constraints

- Specs remain canonical; repository evidence informs reconciliation but never
  replaces Planning intent.
- Every local process/provider call is injected, context-aware, and testable.
- Repository and pull-request identities must agree with the configured target.
- Preview performs no provider, metadata, event, or audit mutation.
- Apply requires `planning.publish`, explicit confirmation, a matching fresh
  plan, and an audit sink.
- Partial remote failure retains completed evidence and performs no destructive
  rollback.
- Last-reconciled evidence is written only after successful provider and mapping
  completion.

## Solution Shape

### Repository evidence adapter

Implement the GitHub repository-evidence capability using canonical provider
identity plus checked-out repository state. Return default/current refs, exact
commit, and an optional planning change request with stable identity, head/base,
draft state, and open/closed/merged classification. Reject detached,
cross-repository, incomplete, or ambiguous evidence when it prevents a safe
decision.

### Application reconciliation service

Add preview and apply services around the existing provider-neutral
`ReconciliationPlan` and `ReconciliationResult`. Preview loads mappings,
repository evidence, and current publication state; then classifies issue-body,
readiness-label, milestone membership, and relationship differences as create,
update, reuse, unchanged, or manual remediation.

Apply re-runs the complete preview, compares repository/provider/mapping
revisions, applies ordered publication actions, persists mapping changes, and
records reconciliation evidence. One event is emitted only when state changed.

### Project boundary

This spec may report that Project workspace evidence needs execution-workspace
handling, but it does not mutate Project fields, items, or status. Those actions
enter the reconciliation result only after the execution-workspace spec lands.

## Flows

### Preview

1. Authorize Planning read.
2. Load revisioned mappings.
3. Resolve repository/ref/commit/change-request evidence.
4. Inspect canonical mapped publication state.
5. Classify safe actions and manual-only drift.
6. Return a deterministic plan without mutation.

### Confirmed apply

1. Require publish permission, confirmation, and audit availability.
2. Re-run preview and compare the complete reviewed plan.
3. Apply ordered safe publication actions.
4. Persist resulting mappings and reconciliation evidence atomically.
5. Emit one reconciliation event when state changed.

## Acceptance Criteria

- Repository evidence returns canonical repository/ref/commit/PR state.
- Cross-repository and ambiguous change-request evidence fails closed.
- Preview classifies issue content, readiness, milestone membership, and
  relationships without writes.
- Apply rejects stale repository, provider, or mapping evidence.
- Partial failures retain completed identities and do not record successful
  reconciliation.
- Successful reconciliation records exact repository evidence and completion
  time.
- Identical reruns perform no provider, mapping, event, or audit mutation.
- Legacy story compatibility can consume shared repository evidence without
  importing GitHub types into Planning application contracts.

## Data / Interfaces

- Existing `RepositoryEvidence`, `ReconciliationPlan`, and
  `ReconciliationResult` contracts remain the provider-neutral boundary.
- GitHub branch, commit, pull-request, issue, and relationship DTOs stay private
  to `planning/github`.
- Existing mapping schema records only the final validated repository evidence
  and reconciliation timestamp; no historical ledger or backfill is added.

## Risks / Open Questions

- No schema migration or backfill is planned. Any implementation that cannot
  preserve current metadata must stop for separate review rather than convert
  user data.
- Detached HEAD, forked pull requests, or multiple candidate pull requests can
  make current change-request identity ambiguous; these cases must fail closed.
- GitHub cannot make all provider writes and the local mapping replacement one
  transaction. Partial evidence and rerun recovery are mandatory.
- Readiness and relationship repair may touch objects changed by users after
  preview; final per-object revision checks must prevent overwrites.
- Project-specific drift remains deliberately deferred to the execution
  workspace spec rather than silently ignored.

## Rollout

1. Implement read-only repository evidence and transcript coverage.
2. Add reconciliation preview with no mutation.
3. Add confirmed publication/mapping apply plus partial recovery.
4. Keep native commands disabled until execution and host-wiring specs finish.
5. Roll back by disabling reconciliation apply; persisted mappings and provider
   objects remain compatible and require no migration.

## Verification

- Fake repository tests for branches, commits, detached state, and PR lifecycle.
- Application tests for preview ordering, confirmation, permissions, stale
  evidence, partial failure, audit failure, and unchanged reruns.
- GitHub transcript tests matching `repository.evidence` and
  `github.reconcile` manifest families.
- Argument/path-space, cancellation, redaction, pagination, Windows, race, and
  full repository matrix checks.

## Dependencies

- [GitHub Coordination Mapping Persistence](./github-coordination-mapping-persistence.md)
- Parent contract: [GitHub Adapter Transition](./github-adapter-transition.md)

## Follow-Up

[GitHub Execution Workspace](./github-execution-workspace.md) adds Project
field, item, drift, and status behavior to reconciliation.

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
