---
created: "2026-04-11T21:53:09Z"
epic: release-install-and-update-flow
project: brain
status: approved
title: Release, Install, And Update Flow Spec
type: spec
updated: "2026-04-16T00:16:00Z"
---
# Release, Install, And Update Flow Spec

Created: 2026-04-11T21:53:09Z

## Why

Brain skills, Brain binaries, and Brain-managed project context must stay version-aligned. If the binary updates but installed skills or project context stay stale, agents miss the current command surface, workflow guidance, and compiler-era context behavior.

## Problem

Binary updates, installed skills, and Brain-managed project context all evolve together. If any one of those surfaces drifts, agents keep stale workflow guidance and older Brain repos silently lag the current command surface. Brain now needs one explicit lifecycle that owns binary install, skill refresh, automatic project soft migrations, observability, and maintainer validation from a branch-built binary.

## Goals

- Bundle the Brain skill into the running binary.
- Make skill installs copy-only for every agent and scope.
- Refresh already-installed global skills during `brain update` and the shell/PowerShell install scripts.
- Refresh already-installed local skills inside the current `--project` during `brain update`.
- Repair stale or legacy local skill installs lazily before project work begins.
- Expose a machine-readable manifest so Brain can detect stale installs without maintaining a machine-wide registry.
- Automatically apply required soft project migrations for the current Brain repo during `brain update`.
- Apply the same soft project migrations lazily the next time Brain runs in another older Brain repo.
- Keep project migrations idempotent, inspectable, and limited to Brain-managed surfaces plus existing agent integration files.

## Non-Goals

- Tracking every local Brain skill install across all repos on disk.
- Tracking every Brain-managed repo on disk or scanning the machine for projects that might need migration.
- Solving the separate Windows `brain.exe` in-place replacement failure from issue #8 inside this same change set.
- Installing new global or local Brain skills automatically for agents that were never configured before.
- Creating a Brain workspace in repos that do not already use Brain.
- Mutating unmanaged user docs or creating missing agent instruction files during automatic migration.

## Requirements

- `brain skills install` and `brain skills targets` must work from any directory, not only from a Brain source checkout.
- Brain skill installs must always be copied directories and must never be symlinked.
- Installed skills must include a generated `.brain-skill-manifest.json` beside `SKILL.md`.
- The manifest must include `schema_version`, `brain_version`, `brain_commit`, `bundle_hash`, `installed_at`, `agent`, and `scope`.
- `bundle_hash` is the authoritative freshness check. Missing or invalid manifests are treated as legacy stale installs.
- `brain update --check` must remain read-only and must not refresh skills or apply project migrations.
- `brain update` must refresh already-installed global skills plus already-installed local skills inside the current `--project`.
- The install scripts must refresh already-installed global skills after installing the binary.
- Local project commands must repair stale local installs before doing work, but only when a local Brain skill is already present in that project.
- Brain must keep a project migration ledger in the repo-local workspace, separate from the skill manifest, so soft migrations are tracked independently from release semver.
- Project migration planning must use named migration ids or steps, not just raw binary version comparison, so multiple Brain releases can share the same migration set when needed.
- `brain update` must inspect the current `--project` and apply pending soft migrations when that directory already uses Brain.
- App-backed Brain commands must lazily apply pending soft migrations before project work begins, similar to the existing local-skill repair path.
- Automatic migrations must be limited to idempotent Brain-managed operations such as refreshing generated context surfaces and updating existing Brain agent integration blocks.
- Automatic migrations must not create a Brain workspace in a non-Brain repo and must not create missing agent-specific instruction files.
- Automatic migrations must preserve user content outside Brain-managed blocks and must leave unmanaged agent files alone unless they already contain a Brain-managed block or legacy Brain wrapper block.
- `brain doctor` must surface whether project soft migrations are current, pending, or broken for the current repo.
- `brain update` JSON output must expose project migration status alongside skill refresh status.

## UX / Flows

- A user can run `brain skills install` or `brain skills targets` from any shell directory because the running binary carries the Brain skill bundle.
- A user who runs `brain update` gets a refreshed binary plus refreshed existing global/current-project local Brain skills in one command.
- A user who updates via `scripts/install.sh` or `scripts/install.ps1` gets the same global skill refresh behavior.
- A user who returns to an older local Brain repo gets an automatic local skill repair before the next app-backed Brain command runs there.
- A user who runs `brain update --project <repo>` inside an existing Brain repo gets any pending soft project migrations applied in the same upgrade flow.
- A user who opens a different older Brain repo later gets the pending soft project migrations lazily on first Brain use in that repo, without any machine-wide scan.
- A user can inspect project migration health through `brain doctor` and can fall back to explicit `brain context refresh --project .` or `brain adopt --project .` remediation when an automatic migration fails.

## Data / Interfaces

- `brain skills install`
  - Remove the public `--mode` flag.
  - Keep output method values as `copy`.
- `brain skills targets`
  - Remove the public `--mode` flag.
- Installed skill directory
  - Add `.brain-skill-manifest.json`.
- Project workspace state
  - Add a project migration ledger under `.brain/state/` with a schema version, applied migration ids, and run metadata.
- `brain update`
  - Add `skill_refresh_status` and `refreshed_skills` to the command JSON payload.
  - Add `project_migration_status` and `applied_project_migrations` to the command JSON payload.
- `brain doctor`
  - Add a `project_migrations` check that reports `current`, `pending`, or `broken`.

## Risks / Open Questions

- The embedded bundle must stay in sync with `skills/brain/` during local development and release builds.
- Update failure messaging must stay explicit when the binary install succeeds but the post-update skill refresh fails.
- Automatic project migration must stay conservative enough that re-running it across active repos does not create noise or clobber user-maintained docs.
- The migration runner should reuse existing idempotent context/adoption primitives where possible so Brain does not grow a second document mutation path.

## Rollout

- Land the embedded bundle plus copy-only installer first.
- Land update/install-script refresh orchestration next.
- Land lazy local self-heal on app-backed commands.
- Land the project migration ledger and first-wave soft migrations next.
- Land automatic project migration during `brain update` and lazy project entry after the runner is proven idempotent.
- Land doctor/update status reporting and remediation messaging with the migration runner.
- Update docs, maintainer workflow notes, and the release/install epic/spec in the same branch.
- Track the separate Windows updater failure as follow-up work.

## Story Breakdown

- Bundle the Brain skill into the running binary and remove cwd-relative skill source resolution.
- Add manifest-based freshness checks and legacy install detection.
- Refresh existing global/current-project local installs during `brain update`.
- Refresh existing global installs in the shell/PowerShell install scripts.
- Repair stale local installs before app-backed commands.
- Add a repo-local project migration ledger and named migration planning model.
- Implement the first-wave automatic soft migrations for generated context and existing agent integration files.
- Run pending project migrations during `brain update` and lazily before app-backed commands in older Brain repos.
- Surface migration status, failure handling, and remediation in `brain update` and `brain doctor`.
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
- Use the project migration ledger as the source of truth for soft repo upgrades. Do not overload the skill manifest for project migration state, because skill freshness and project migration state have different lifecycles.
- When a branch changes automatic project-upgrade behavior, validate that migration path from the branch-built binary against a representative older Brain repo before merge, just like unreleased Brain skill changes are validated from the current branch binary.
