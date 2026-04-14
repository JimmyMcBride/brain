---
created: "2026-04-11T05:10:54Z"
project: brain
spec: release-install-and-update-flow
title: Release, Install, And Update Flow
type: epic
updated: "2026-04-14T05:45:49Z"
---
# Release, Install, And Update Flow

Created: 2026-04-11T05:10:54Z

## Description

Own the entire Brain binary plus Brain skill lifecycle so installs and updates stay version-aligned without manual repo checkouts or symlinked skill directories.

## Work Items

- Bundle the Brain skill into the running binary and make `brain skills install` copy-only.
- Write a machine-readable skill manifest beside installed `SKILL.md` files and use the bundle hash as the freshness signal.
- Refresh already-installed global skills during `brain update` and both install scripts.
- Refresh already-installed local skills inside the current `--project` during `brain update`.
- Repair stale legacy local project installs lazily before app-backed commands run.
- Update maintainer scripts and docs to validate unreleased skill changes with a current-branch binary instead of a repo-root-relative install source.
- Track the separate Windows in-place binary replacement failure as follow-up work outside this epic.

## Notes

- `brain skills install` and `brain skills targets` must work from any directory, not just a Brain source checkout.
- Existing local installs in other repos should not require a machine-wide registry; they repair themselves the next time Brain runs there.
- Installed skills should never be symlinked because the binary and skill content must move together across updates.

## Spec

- [[.brain/planning/specs/release-install-and-update-flow.md]]
