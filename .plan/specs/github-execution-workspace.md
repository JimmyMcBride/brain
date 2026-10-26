---
created_at: "2026-10-26T14:28:47Z"
initiative: github-adapter-transition
project: brain
slug: github-execution-workspace
status: done
title: GitHub Execution Workspace
type: spec
updated_at: "2026-10-26T18:59:15Z"
---

# GitHub Execution Workspace

Created: 2026-10-26T14:28:47Z

## Why

Creating or connecting a GitHub Project establishes workspace identity but does
not attach published issues or reflect Planning execution status. Users need a
reviewed, idempotent path from canonical Planning work to Project items and
semantic delivery status.

## Problem

Provider-neutral execution-workspace contracts exist, while GitHub inspect,
attach, and status operations remain unsupported. Project field IDs, item IDs,
and regenerated single-select option IDs are provider details that must stay
inside the adapter without weakening confirmation or drift handling.

## Goals

- Inspect one canonical Project workspace, supported fields, and tracked items.
- Attach published Planning issues without duplicate Project items.
- Apply `todo`, `in_progress`, `in_review`, or `done` through semantic status
  mapping.
- Detect field, item, and status drift during check/reconciliation.
- Require explicit confirmation and `planning.execute` for status mutation.
- Return manual remediation when safe option mapping is unavailable.

## Non-Goals

- Creating arbitrary Project fields or status options.
- Deleting Project items or user-owned fields.
- Supporting custom status vocabularies beyond the retained compatibility map.
- Making GitHub Project state canonical.
- Native command/config wiring.
- General Project automation outside Planning artifacts.

## Constraints

- GitHub node, field, item, and option IDs remain adapter-private.
- Project identity must match the persisted reviewed workspace reference.
- All connections use bounded pagination and reject incomplete evidence.
- Title-only workspace or item matching is forbidden.
- Existing single-select options may be used only after unambiguous semantic
  validation; missing/regenerated options require manual remediation.
- Attach and unchanged status transitions emit no duplicate write or event.
- Final provider revisions are returned for reconciliation and stale-plan checks.

## Solution Shape

### Workspace inspection

Implement the execution-workspace inspection port using GraphQL ProjectV2
evidence. Validate canonical owner/number/node/URL identity, locate the supported
status field by retained semantic convention, normalize recognized options to
provider-neutral statuses, and return tracked published-work items with stable
references and current status.

### Item attachment

Resolve the published issue by stable node identity. Reuse an existing Project
item when the content identity matches; otherwise add it once and return final
item evidence. Reject duplicate/ambiguous item evidence or repository mismatch.

### Status application

Re-inspect the target item and status field immediately before mutation. Apply
only the reviewed semantic transition using the validated current option ID.
Return unchanged without mutation when already at target status. Missing fields,
options, or membership produce typed manual remediation rather than hidden
field creation.

### Application integration

Add preview/confirmed execution-status services and integrate execution actions
with reconciliation. Planning owns allowed semantic status and action
classification; the adapter owns GitHub representation. One execution-status
event is emitted only after a real changed transition.

## Flows

### Attach

1. Resolve persisted workspace and published issue references.
2. Inspect complete bounded Project item state.
3. Return existing item or preview one attach action.
4. Revalidate and attach once after confirmation where the host workflow
   requires mutation approval.

### Status transition

1. Inspect workspace fields, options, item membership, and current status.
2. Preview semantic transition.
3. Require execution permission, confirmation, and audit sink.
4. Re-inspect revision-bound evidence.
5. Apply one status mutation or return unchanged.

## Acceptance Criteria

- Inspection exposes only provider-neutral workspace, field, item, and status
  values.
- Existing item attachment is reused without mutation.
- New attachment happens once and returns stable evidence.
- All four retained statuses map correctly when supported.
- Missing, duplicate, regenerated, or unsupported options fail with manual
  remediation and no mutation.
- Stale workspace/item/field evidence prevents writes.
- Reconciliation reports Project drift and applies only reviewed safe actions.
- Identical attach/status/reconcile reruns are write-free and event-free.
- Disabled adapter performs zero provider or filesystem work.

## Data / Interfaces

- Existing `ExecutionWorkspaceSnapshot`, `ExecutionWorkItem`,
  `ExecutionAttachRequest`, and `ExecutionStatusRequest` remain the shared
  contract.
- Provider-private GraphQL DTOs carry Project, field, option, and item IDs.
- No new `.plan/` schema is introduced; stable workspace identity continues
  through the existing GitHub metadata representation.

## Risks / Open Questions

- No schema migration or backfill is planned. Stable workspace mappings must
  remain readable before, during, and after rollback.
- GitHub can regenerate single-select option IDs while retaining labels. The
  adapter must revalidate semantics and require manual remediation when mapping
  is uncertain.
- Large Projects can exceed bounded field/item listings. Incomplete evidence
  must stop mutation instead of guessing.
- A lost attach/status response can leave remote state changed. Rerun must
  inspect before retrying and avoid duplicate items or transitions.
- User-owned Project customization must remain untouched; this spec supports
  only retained Planning semantics.

## Rollout

1. Land read-only Project field/item inspection.
2. Add idempotent item attachment.
3. Add previewed status transitions and reconciliation integration.
4. Keep native commands disabled until host rollout.
5. Roll back by disabling execution mutations; Project identity mappings and
   existing items remain readable without backfill.

## Verification

- Fake GraphQL coverage for user/org Projects, fields, options, items,
  pagination, duplicates, and identity mismatch.
- Attach/status tests for changed, unchanged, stale, partial, and lost-response
  behavior.
- Manifest replay for `github.project.status`, `check.github`,
  `project-read`, and `project-status` transcripts.
- Permission, confirmation, audit, redaction, cancellation, race, Windows, and
  full repository checks.

## Dependencies

- [GitHub Coordination Mapping Persistence](./github-coordination-mapping-persistence.md)
- [GitHub Repository Reconciliation](./github-repository-reconciliation.md)
- Parent contract: [GitHub Adapter Transition](./github-adapter-transition.md)

## Follow-Up

[Native GitHub Planning Rollout](./native-github-planning-rollout.md) exposes
the completed capabilities through Brain.

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
