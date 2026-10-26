---
created_at: "2026-10-26T14:28:47Z"
initiative: github-adapter-transition
project: brain
slug: github-integration-conformance-release
status: implementing
title: GitHub Integration Conformance and Release
type: spec
updated_at: "2026-10-26T20:40:18Z"
---

# GitHub Integration Conformance and Release

Created: 2026-10-26T14:28:47Z

## Why

Phase 5 completes only when Brain and standalone Plan deliver one proven GitHub
workflow and users can install verified releases confidently.

## Problem

Brain and standalone Plan can appear individually complete while behaving
differently for the same GitHub-backed workflow. Users need one verified,
installable transition where both entrypoints preserve their documented
contracts and recovery behavior.

## Goals

- Prove Brain and standalone Plan deliver the same retained GitHub outcomes.
- Publish installable, verified releases in safe order.
- Move shared behavior into Brain while preserving documented standalone-only
  compatibility.
- Keep rollback possible without losing user planning data.
- Finalize support and transition guidance.

## Non-Goals

- Removing GitHub source mode or setting a deprecation date.
- Moving legacy source ownership or story creation into Brain.
- Starting Linear cleanup, the Brain knowledge loop, cloud, or hybrid sync.
- Shipping unrelated roadmap phases in the same rollout.

## Constraints

- Brain releases first; Plan pins an immutable stable tag.
- Standalone Plan uses no `replace` directive or unpublished Brain revision.
- Each repository keeps an independently reviewable and revertible PR.
- Required conformance uses copied fixtures and injected fake providers.
- Intentional differences from Plan `v0.1.29` stay documented: preview,
  confirmation, safe unchanged reruns, and stronger failure recovery.
- Release artifacts include checksums and Linux, macOS, and Windows archives for
  amd64 and arm64.
- Rollback preserves `.plan/` artifacts and `.plan/.meta/github.json` mappings.

## Solution Shape

### Brain conformance gate

Run every migrated manifest family through native Brain commands against
independent fixtures. Compare exit class, normalized human output, JSON shape,
provider transcript, file effects, remote identities, metadata, partial
evidence, and identical rerun. Verify dependency/export guards and disabled-mode
zero-work behavior.

### Brain release

Merge the complete Brain implementation, run full local and CI matrices, tag a
stable release from the protected release flow, publish checksums and six
platform archive pairs, and record the immutable module version for Plan.

### Plan wrapper transition

Pin the stable Brain module, route mapped remote discuss, GitHub administration,
check, guide, reconciliation, and Project-status families through shared
packages, and retain standalone presentation/authorization plus explicitly
compatibility-only source/story behavior.

### Final release and support docs

Replay manifest and full matrix in Plan, publish its release, then update Brain
and Plan transition docs with supported commands, enablement, permissions,
failure recovery, compatibility boundaries, and Phase 5 completion status.

## Flows

### Coordinated release

1. Complete all prerequisite Brain specs.
2. Replay native Brain conformance and full matrix.
3. Publish stable Brain tag and verify assets/checksums.
4. Open Plan PR pinning that tag and routing mapped commands.
5. Replay Plan conformance and full matrix without local replacement.
6. Publish Plan and verify assets/checksums.
7. Update transition docs and mark Phase 5 complete.

### Rollback

1. Disable native adapter wiring or revert the affected host wrapper.
2. Retain artifacts and mappings unchanged.
3. Restore Plan-owned compatibility behavior where required.
4. Re-run conformance before republishing.

## Acceptance Criteria

- Every migrated manifest family passes through Brain native commands.
- Brain stable release contains complete GitHub adapter/native surface.
- Plan pins the stable Brain tag with no `replace` directive.
- Mapped Plan commands use shared Brain implementation and preserve declared
  compatibility output/effects.
- Both repositories pass tests, race, vet, build, module verification, Windows
  checks, Plan checks, and release workflows.
- Brain and Plan releases publish checksums plus Linux/macOS/Windows amd64/arm64
  archives.
- Disabled adapter and local-only Planning remain functional without `gh`.
- Documentation names retained compatibility-only behavior and confirms GitHub
  source support through Phase 9 plus a separate deprecation decision.
- Phase 5 roadmap/spec status changes only after release evidence is verified.

## Data / Interfaces

- Brain publishes one immutable Go module version consumed by standalone Plan.
- Existing workspace and GitHub metadata schemas remain unchanged through the
  coordinated release.
- Release outputs retain current archive/checksum naming and platform matrix.
- Conformance comparisons use the versioned GitHub v1 manifest and copied
  fixtures rather than live provider state.

## Risks / Open Questions

- No workspace or GitHub metadata migration/backfill is planned. Release or
  rollback must preserve current files byte-compatibly except for reviewed
  operational updates.
- Brain/Plan release skew can leave mapped commands on incompatible contracts;
  Plan must pin only after Brain assets and module resolution are verified.
- Cross-platform archive or checksum failure must stop publication completion,
  even when source tests pass.
- Compatibility-only commands can accidentally migrate or warn too early;
  disposition checks must remain explicit.
- No unresolved support-policy choice remains: GitHub source mode continues
  through Phase 9 and a separate deprecation decision.

## Rollout

1. Replay Brain native conformance and full matrix.
2. Publish and verify stable Brain release.
3. Land Plan wrapper PR pinned to that exact release.
4. Replay Plan conformance/full matrix and publish Plan.
5. Update docs and mark Phase 5 complete only after both releases verify.
6. Rollback restores prior wrapper/native routing without deleting or migrating
   user artifacts and mappings.

## Verification

- Full manifest replay against independent fake-provider fixtures in both hosts.
- Exact checks for partial failure, state-write failure, stale revision,
  ambiguity, pagination, Project drift/options, cancellation, and unchanged
  reruns.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`,
  Windows compilation/tests, `go mod verify`, and Plan analysis/checklists in
  both repositories.
- Release asset count, platform naming, checksum, tag, and module-resolution
  verification.

## Dependencies

- [GitHub Coordination Mapping Persistence](./github-coordination-mapping-persistence.md)
- [GitHub Repository Reconciliation](./github-repository-reconciliation.md)
- [GitHub Execution Workspace](./github-execution-workspace.md)
- [Native GitHub Planning Rollout](./native-github-planning-rollout.md)
- Parent contract: [GitHub Adapter Transition](./github-adapter-transition.md)

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
