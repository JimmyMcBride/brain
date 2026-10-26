---
created_at: "2026-10-26T14:28:47Z"
initiative: github-adapter-transition
project: brain
slug: github-coordination-mapping-persistence
status: approved
title: GitHub Coordination Mapping Persistence
type: spec
updated_at: "2026-10-26T15:12:28Z"
---

# GitHub Coordination Mapping Persistence

Created: 2026-10-26T14:28:47Z

## Why

GitHub publication can now create or reuse milestones and Projects, but later
Planning runs cannot always recover those choices. Coordination must survive a
completed workflow so users can continue safely without repeating setup.

## Problem

Coordination created during publication is not reliably available to later
Planning runs. Users can finish remote publication successfully, then discover
that the next run cannot recover the milestone or Project chosen during the
reviewed workflow. Repeating work risks ambiguity, manual repair, or duplicate
provider objects.

## Goals

- Remember resulting milestone and Project identities after confirmed work.
- Preserve existing workspace compatibility and user-owned information.
- Trust only complete, validated results.
- Recover when GitHub succeeds but local recording fails.
- Keep identical reruns unchanged.

## Non-Goals

- Configuring Project fields, items, or execution status.
- Implementing repository reconciliation.
- Changing source ownership or migrating existing workspaces.
- Recording an external identity for a reviewed `skip` decision, because none
  exists.
- Adding native GitHub commands.

## Constraints

- Provider mutation completes before mapping persistence.
- Only completed action evidence may add or replace an external identity.
- A failed action reference may be retained in failure evidence but cannot be
  silently promoted into successful mapping state.
- Group identity applies only to the reviewed complete group membership.
- Workspace identity must remain unique and canonical for the target repository.
- Stale mapping revisions fail before local replacement.
- Existing private compatibility fields and unknown metadata remain preserved.

## Solution Shape

### Evidence-to-mapping merge

Extend the application-owned adoption/persistence path to merge stable
references from the confirmed publication result after provider apply. Match
each evidence entry to the reviewed action in order and reject extra, missing,
reordered, or identity-changing evidence.

Artifact references retain current behavior. A completed group action maps its
canonical milestone identity to every reviewed member artifact. A completed
non-skipped workspace action records one canonical Project identity. Reuse and
unchanged actions retain their known identities without provider writes.

### GitHub metadata persistence

Teach the GitHub mapping adapter to encode reviewed workspace identity changes
through the existing project-decision representation. Preserve unrelated
legacy-private fields, existing titles/reasons where the provider-neutral view
cannot replace them, and the current file schema. Reject cross-repository,
malformed, duplicate, or conflicting Project identities.

### Recovery and idempotency

If provider apply succeeds and mapping save fails, return publication evidence
plus the mapping failure without rollback. The next preview re-inspects remote
state, classifies the provider objects as reuse or unchanged, and retries only
the missing mapping replacement. A fully persisted rerun performs no mutation
and emits no event.

## Flows

### Confirmed adoption

1. Load revisioned mappings and inspect provider state.
2. Preview the complete publication and mapping intent.
3. Revalidate confirmation, permissions, provider revisions, and mapping
   revision.
4. Apply ordered provider actions.
5. Merge completed artifact, group, and workspace evidence.
6. Atomically save the complete mapping state.
7. Emit one adoption event only when provider or mapping state changed.

### Save failure recovery

1. Provider apply returns durable identities.
2. Mapping save fails or loses its response.
3. Result retains completed provider evidence and reports partial failure.
4. Rerun recovers provider objects by stable identity and managed source
   evidence.
5. Rerun writes only missing mapping state.

## Acceptance Criteria

- New and reused milestone identities persist for every reviewed group member.
- Created and connected Project identities persist canonically.
- A skipped workspace produces no fabricated external reference.
- Mapping saves accept valid workspace changes and reject ambiguous or foreign
  Projects.
- Existing GitHub metadata fixtures remain readable without migration.
- State-write failure after remote success preserves recovery evidence.
- Identical reruns leave provider calls, mapping bytes, events, and audit output
  unchanged.
- Local Planning and disabled GitHub adapter behavior remain unchanged.

## Data / Interfaces

- `ExternalMappingState.GroupReferences` and `WorkspaceReferences` remain the
  provider-neutral durable view.
- `.plan/.meta/github.json` retains its current schema and observable field
  layout; no backfill is introduced.
- Existing provider action/result DTOs remain version 1 unless implementation
  proves an additive evidence field is required.

## Risks / Open Questions

- Integration-contract ownership crosses application orchestration and the
  GitHub persistence adapter; neither side may infer evidence the other did not
  validate.
- Legacy Project decisions contain private choice/title/reason fields that are
  richer than the provider-neutral reference view. Persistence must preserve
  those fields without inventing values.
- One completed group identity maps across several artifacts; partial evidence
  must never leave only a subset looking authoritative.
- A lost mapping-save response may mean bytes were committed. Recovery must
  reread before retrying rather than assume failure.
- No unresolved product choice remains; implementation should stop if current
  metadata cannot encode one canonical workspace without shape change.

## Rollout

1. Add application evidence-merge tests before changing persistence.
2. Enable group persistence, then workspace persistence against copied legacy
   fixtures.
3. Replay create/connect/skip, partial failure, and identical rerun cases.
4. Roll back by reverting the evidence merge/save support; existing metadata
   remains readable because no schema migration or backfill occurs.

## Verification

- Application tests for evidence ordering, group membership, workspace choices,
  partial results, and stale mapping revisions.
- GitHub adapter fixtures for create/connect/skip, legacy-private field
  preservation, cross-repository rejection, and semantic no-op saves.
- Lost-response and local-save-failure tests proving no duplicate remote
  objects.
- Full test, race, vet, build, module verification, Windows compilation, and
  Phase 5 Plan checks.

## Dependencies

- Parent contract: [GitHub Adapter Transition](./github-adapter-transition.md)
- Preceded by merged milestone and Project publication identity support.

## Follow-Up

[GitHub Repository Reconciliation](./github-repository-reconciliation.md)
consumes the persisted identities.

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
