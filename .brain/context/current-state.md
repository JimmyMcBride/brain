---
updated: "2026-04-12T03:19:46Z"
---
# Current State

<!-- brain:begin context-current-state -->
This file is a deterministic snapshot of the repository state at the last refresh.

## Repository

- Project: `brain`
- Root: `.`
- Runtime: `go`
- Go module: `brain`
- Current branch: `docs-pr-workflow-and-windows-ci-fixes`
- Remote: `https://github.com/JimmyMcBride/brain.git`
- Go test files: `19`

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
- 2026-04-11: Fixed publish-only session closeout so finish validation can treat accepted durable notes committed in the session commit range as satisfying the memory rule, and ignore volatile `.brain/state` plus session runtime files when checking meaningful git cleanliness.
- 2026-04-11: Fixed automatic GitHub release publishing so main pushes no longer stop at tag creation. The tag workflow now calls the reusable release workflow directly after pushing the new semver tag, which avoids the GitHub `GITHUB_TOKEN` workflow-chaining limitation that was leaving tags without downloadable release assets.
- 2026-04-11: Updated maintainer docs to make feature-branch plus PR merge the default release path. The documented flow is now branch -> verify -> commit -> PR -> merge to main -> automatic release -> local global-brain refresh, while direct pushes to main remain possible but are treated as the exception.
- 2026-04-11: Fixed the Windows CI failures by making config tests OS-aware, using `errors.Is(..., os.ErrNotExist)` for epic-spec migration so Windows missing-file errors backfill specs correctly, normalizing projectcontext golden comparisons across line endings, and adding a test-only writable-target hook so updater fallback tests do not depend on Unix directory permission semantics.
- 2026-04-11: Enabled GitHub branch protection for `main` via `gh api` so direct pushes are blocked, admins are enforced, PRs are required, and the `test (ubuntu-latest)` plus `test (windows-latest)` checks must pass before merge. Also narrowed `ci.yml` to run on `pull_request` and on pushes to `main` only, which removes duplicate branch-push CI runs.
- 2026-04-11: Fixed the last Windows-only plan test failure on the PR branch by replacing the Unix-specific missing-file string check in epic-spec migration with `errors.Is(err, os.ErrNotExist)`, so legacy epic/story backfill now creates the canonical spec correctly on Windows too.
- 2026-04-11: Relaxed `main` branch protection just enough to allow normal PR merges again by disabling `required_linear_history` while keeping required PRs, required CI checks, and admin enforcement. Also updated the release workflow to set the GitHub Release title explicitly to `${RELEASE_TAG}` so releases are named by version instead of showing `main`.
- 2026-04-11: Enabled GitHub `delete_branch_on_merge` for this repo, so merged feature branches will now be deleted automatically after PR merge unless GitHub cannot remove the branch.
- 2026-04-11: Simplified CI policy again so validation now runs only on `pull_request`. Merges to `main` no longer trigger the separate CI workflow; after merge, only the release automation runs. Branch protection still requires the PR checks before merge.
- 2026-04-11: Fixed the reusable GitHub release workflow so auto-releases no longer resolve to `main`. The workflow now resolves tag and sha explicitly in a dedicated job and publishes the GitHub Release plus assets against the semver tag passed in from the main tagger workflow.
- 2026-04-11: Stopped generating agent wrapper files implicitly from installed global skills. Project context now creates wrappers only when `--agent` is passed, removed the tracked `.codex/`, `.claude/`, and `.openclaw/` wrapper files from this repo, and deleted dead wrapper reference files under `skills/`.
- 2026-04-11: Corrected the reusable GitHub release workflow again after confirming the called workflow still saw the parent `push` event context. Release tag resolution now prefers non-empty workflow-call inputs over `github.event_name`, so auto releases should publish `vX.Y.Z` assets and release names instead of `main`.
- 2026-04-11: Merged the latest `origin/main` into the PR branch while resolving conflicts in `.github/workflows/release.yml` and `.brain/context/current-state.md`. Kept the newer workflow-call input preference for release tag resolution and preserved the latest durable note entries.
