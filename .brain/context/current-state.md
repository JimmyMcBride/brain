---
updated: "2026-04-11T22:10:20Z"
---
# Current State

<!-- brain:begin context-current-state -->
This file is a deterministic snapshot of the repository state at the last refresh.

## Repository

- Project: `brain`
- Root: `.`
- Runtime: `go`
- Go module: `brain`
- Current branch: `main`
- Remote: `https://github.com/JimmyMcBride/brain.git`
- Go test files: `18`

## Docs

- `README.md`
- `docs/architecture.md`
- `docs/project-architecture.md`
- `docs/project-overview.md`
- `docs/project-workflows.md`
- `docs/skills.md`
- `docs/usage.md`
- `docs/why.md`
<!-- brain:end context-current-state -->

## Local Notes

Add repo-specific notes here. `brain context refresh` preserves content outside managed blocks.

- 2026-04-11: Hardened note updates to normalize full-note stdin/frontmatter safely and tightened `brain skills` so it now installs only the Brain skill instead of acting like a multi-skill bundle installer.
- 2026-04-11: Installed the updated global `brain` binary from commit `93e71a6` and pushed the note-integrity plus multi-skill install changes to `main`.
- 2026-04-11: Added the brain emoji to the README title and published the change to `main`.
- 2026-04-11: Rewrote the README and why-doc wording to describe Brain in the present tense without historical framing.
- 2026-04-11: Added retrieval observability with tracked index freshness metadata, `brain search status`, `brain search --explain`, and a doctor check for stale or missing local index state.
- 2026-04-11: Updated the Brain skill guidance to use `brain search status`, `brain search --explain`, and `doctor` index freshness when debugging retrieval.
- 2026-04-11: Updated the global `brain` binary to commit `f741dea` and refreshed the global Codex `brain` skill so it includes retrieval freshness and explain/status guidance.
- 2026-04-11: Added the roadmap epic `Core Product Tightening And Simplification` with six tracking stories for policy correctness, retrieval truth, adoption, context duplication, session defaults, and find behavior.
- 2026-04-11: Fixed `.brain/policy.override.yaml` merge semantics so boolean policy fields can be explicitly turned both on and off, and added projectcontext tests for both directions.
- 2026-04-11: Tightened the retrieval language in the README, usage docs, and Brain skill so the default `localhash` provider is described honestly as lexical search plus lightweight semantic hinting, while doctor/search status remain the source of truth for the active provider.
- 2026-04-11: Fixed the session manager race on `.brain/session.json` by serializing active-session mutations with a filesystem lock, writing session state atomically, and making `brain session run` refuse to record into a session that was finished or aborted while the command was still running.
- 2026-04-11: Installed the global `brain` binary from commit `7cc80a2` with embedded build metadata and prepared the product-tightening roadmap plus session concurrency fixes for push to `main`.
- 2026-04-11: Added `scripts/refresh-global-brain.sh` plus a maintainer reference note so repo maintainers can rebuild `~/.local/bin/brain` with embedded build metadata and sync the global Codex `brain` skill from `skills/brain/` without treating the refresh itself as a new product change.
- 2026-04-11: Published the maintainer-only global refresh workflow and script so maintainers can rebuild `~/.local/bin/brain`, sync the global Codex `brain` skill, and verify both against the pushed repo `HEAD`.
- 2026-04-11: Added `brain adopt` as the first-class existing-repo onboarding path. It shares bootstrap logic with `brain init`, adopts Brain-owned markdown into the managed-block model, reports `adopted` for unmanaged files, and preserves previous content under `Local Notes` instead of clobbering it.
- 2026-04-11: Published `brain adopt` as the existing-repo onboarding path and pushed the shared bootstrap plus managed-file adoption workflow to `main`, then refreshed the installed binary and global Codex `brain` skill from that release state.
- 2026-04-11: Added Windows support across config pathing, release assets, `brain update`, a PowerShell installer, a PowerShell maintainer refresh script, Windows CI/release coverage, and the user-facing install/update docs. Windows now targets `%LocalAppData%\\Programs\\brain\\brain.exe` by default and uses `.zip` release assets with checksum verification.

- 2026-04-11: Recorded the Windows support rollout through a Brain-managed current-state update so session closeout tracks the repo change.

- 2026-04-11: Added first-class Copilot and Pi skill targets based on their documented roots. Copilot now installs globally to `~/.copilot/skills` and locally to `.github/skills`; Pi now installs globally to `~/.pi/agent/skills` and locally to `.pi/skills`.

- 2026-04-11: Simplified `brain skills` so it now installs only the Brain skill, removed the repo-owned `googleworkspace-cli` bundle, and rewrote the README/usage docs around adding the Brain skill globally or locally.

- 2026-04-11: Updated `scripts/refresh-global-brain.sh` to match the new Brain-only `brain skills install` CLI so maintainer refreshes no longer rely on the removed `--skill` flag.
- 2026-04-11: Replaced the old paradigm-based planning model with an opinionated epic-only spec-driven workflow. `brain plan` now centers on brainstorm -> epic -> spec -> stories, removes milestone/cycle support, auto-creates one canonical draft spec per epic, gates new stories on approved specs, and migrates legacy epic projects by backfilling spec notes and story metadata.
- 2026-04-11: Removed the final legacy planning compatibility path from `.brain/project.yaml` so Brain now accepts only `planning_model: epic_spec_v1`, and rewrote the README flow so install is followed immediately by adding the Brain skill and then by the brainstorm -> epic -> spec -> story execution workflow.
- 2026-04-11: Added automatic stable release tagging from `main` so each push creates the next patch semver tag, triggers the existing GitHub release packaging flow, and makes installers plus `brain update` target the latest stable release from `main` by default.
