---
created: "2026-04-11T21:53:09Z"
epic: release-install-and-update-flow
project: brain
status: draft
title: Release, Install, And Update Flow Spec
type: spec
updated: "2026-04-14T05:45:49Z"
---
# Release, Install, And Update Flow Spec

Created: 2026-04-11T21:53:09Z

## Why

Brain skills and Brain binaries must stay version-aligned. If the binary updates but installed skills stay stale, agents miss the current command surface and workflow guidance.

## Problem

The old `brain skills` flow resolved `skills/brain` from the current working directory, did not refresh installed skills during `brain update`, and allowed symlinked installs that could drift away from the installed binary.

## Goals

- Bundle the Brain skill into the running binary.
- Make skill installs copy-only for every agent and scope.
- Refresh already-installed global skills during `brain update` and the shell/PowerShell install scripts.
- Refresh already-installed local skills inside the current `--project` during `brain update`.
- Repair stale or legacy local skill installs lazily before project work begins.
- Expose a machine-readable manifest so Brain can detect stale installs without maintaining a machine-wide registry.

## Non-Goals

- Tracking every local Brain skill install across all repos on disk.
- Solving the separate Windows `brain.exe` in-place replacement failure from issue #8 inside this same change set.
- Installing new global or local Brain skills automatically for agents that were never configured before.

## Requirements

- `brain skills install` and `brain skills targets` must work from any directory, not only from a Brain source checkout.
- Brain skill installs must always be copied directories and must never be symlinked.
- Installed skills must include a generated `.brain-skill-manifest.json` beside `SKILL.md`.
- The manifest must include `schema_version`, `brain_version`, `brain_commit`, `bundle_hash`, `installed_at`, `agent`, and `scope`.
- `bundle_hash` is the authoritative freshness check. Missing or invalid manifests are treated as legacy stale installs.
- `brain update --check` must remain read-only and must not refresh skills.
- `brain update` must refresh already-installed global skills plus already-installed local skills inside the current `--project`.
- The install scripts must refresh already-installed global skills after installing the binary.
- Local project commands must repair stale local installs before doing work, but only when a local Brain skill is already present in that project.

## UX / Flows

- A user can run `brain skills install` or `brain skills targets` from any shell directory because the running binary carries the Brain skill bundle.
- A user who runs `brain update` gets a refreshed binary plus refreshed existing global/current-project local Brain skills in one command.
- A user who updates via `scripts/install.sh` or `scripts/install.ps1` gets the same global skill refresh behavior.
- A user who returns to an older local Brain repo gets an automatic local skill repair before the next app-backed Brain command runs there.

## Data / Interfaces

- `brain skills install`
  - Remove the public `--mode` flag.
  - Keep output method values as `copy`.
- `brain skills targets`
  - Remove the public `--mode` flag.
- Installed skill directory
  - Add `.brain-skill-manifest.json`.
- `brain update`
  - Add `skill_refresh_status` and `refreshed_skills` to the command JSON payload.

## Risks / Open Questions

- The embedded bundle must stay in sync with `skills/brain/` during local development and release builds.
- Update failure messaging must stay explicit when the binary install succeeds but the post-update skill refresh fails.

## Rollout

- Land the embedded bundle plus copy-only installer first.
- Land update/install-script refresh orchestration next.
- Land lazy local self-heal on app-backed commands.
- Update docs, maintainer workflow notes, and the release/install epic/spec in the same branch.
- Track the separate Windows updater failure as follow-up work.

## Story Breakdown

- Bundle the Brain skill into the running binary and remove cwd-relative skill source resolution.
- Add manifest-based freshness checks and legacy install detection.
- Refresh existing global/current-project local installs during `brain update`.
- Refresh existing global installs in the shell/PowerShell install scripts.
- Repair stale local installs before app-backed commands.
- Update docs, maintainer scripts, and repo workflow guidance.

## Resources

- [[.brain/planning/epics/release-install-and-update-flow.md]]
- [[docs/skills.md]]
- [[docs/usage.md]]
- [[docs/project-workflows.md]]
- [[scripts/install.sh]]
- [[scripts/install.ps1]]
- [[scripts/refresh-global-brain.sh]]
- [[scripts/refresh-global-brain.ps1]]

## Notes

- Use the running binary as the source of truth for installed Brain skill content. Unreleased skill changes should be validated with `go run .` or another branch-built binary instead of relying on the repo checkout path.
