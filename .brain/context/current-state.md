---
updated: "2026-04-16T04:52:53Z"
---
# Current State

<!-- brain:begin context-current-state -->
This file is a deterministic snapshot of the repository state at the last refresh.

## Repository

- Project: `brain`
- Root: `.`
- Runtime: `go`
- Go module: `brain`
- Current branch: `feature/context-packet-optimization`
- Default branch: `main`
- Remote: `https://github.com/JimmyMcBride/brain.git`
- Go test files: `27`

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

- 2026-04-16: Refined the parked capsules plan so it stays on the shelf cleanly. If `derived-doc-capsules-and-drift-audit` is revived later, the intended operator-facing shape is `capsules=off|auto|on`, default `off`, with `auto` making per-compile decisions from local fresh-packet telemetry and capsule health rather than silently flipping a permanent global switch.

- 2026-04-16: Added a small fresh-packet telemetry slice to `brain context stats`. Local compiler telemetry now rolls up fresh packet budget pressure separately from reused and delta packets, reports how often fresh packets were under pressure, and highlights recurring omitted markdown docs so the capsules decision can stay evidence-based instead of speculative.

- 2026-04-16: Evaluated whether session reuse already cuts enough repeated packet weight to defer capsules. Using branch code via `go run .` on representative tasks, fresh compile responses were about `1425-1431` human-estimated tokens and `2781-2787` JSON-estimated tokens with packet budgets around `891-897 / 900`, while repeated same-task compiles dropped to about `281-287` human-estimated tokens and `412-418` JSON-estimated tokens as compact `reused` responses, roughly an `80%` human reduction and `85%` JSON reduction. A clean same-task fingerprint change probe produced a compact `delta` response at about `283` human-estimated tokens and `421` JSON-estimated tokens with explicit invalidation reasons. The current recommendation is to hold `derived-doc-capsules-and-drift-audit` until real evidence shows the first-turn full packet, not repeated packet weight, is still the bottleneck.

- 2026-04-16: Completed the second context-packet-optimization execution slice for session packet reuse. `brain context compile` now fingerprints relevant compile inputs such as task, budget, changed files, touched boundaries, durable search signals, source summary state, and verification requirements; reuses the latest matching active-session packet as a compact response when those inputs are unchanged; emits compact `delta` responses with changed sections, changed item ids, and invalidation reasons when the task is stable but the packet changed; supports `--fresh` to bypass reuse; stores full packet bodies plus lineage metadata in session records; and surfaces cache status plus reuse or delta lineage in `brain context explain`, `docs/usage.md`, and `skills/brain/SKILL.md`.

- 2026-04-15: Ran a real-task calibration pass over the new budgeted context packet presets and found that the main issue was budget accounting, not the preset constants themselves. The compiler was double-counting note provenance in `budget.used` for note-bearing packets and was not reserving the fixed working-set plus provenance section overhead before optional selection. That follow-up is now fixed, focused tests enforce that representative packets stay within target budgets, and the sampled `small|default|large` presets now behave like lean / normal / roomy tiers on representative Brain tasks without further constant changes.

- 2026-04-15: Completed the first context-packet-optimization execution slice for budgeted context packets. `brain context compile` now supports deterministic `small|default|large` presets plus explicit integer token budgets, all compiler-facing and compiled packet items expose estimated token costs, working-set selection now omits optional boundaries/files/tests/notes under a hard remaining budget instead of fixed item-count caps alone, and both compile and explain surfaces now report budget target, used, remaining, reserve buckets, and top omitted candidates. `docs/usage.md` plus `skills/brain/SKILL.md` were updated in the same branch, and representative tests now pin that tighter presets actually emit leaner working sets while keeping mandatory sections.

- 2026-04-15: Started a new planning branch from fresh `origin/main` for the next context-efficiency layer. Added the new brainstorm note `.brain/brainstorms/token-efficient-context-direction.md` plus three draft epic/spec pairs for `budgeted-context-packets`, `session-packet-reuse`, and `derived-doc-capsules-and-drift-audit`, all framed around the 20/80 token-efficiency path: hard packet budgets first, session-local packet reuse second, and derived doc capsules with drift auditing instead of an always-injected rule-file model.

- 2026-04-16: Ran the full planning loop for the next context-efficiency layer. Approved the three specs for `budgeted-context-packets`, `session-packet-reuse`, and `derived-doc-capsules-and-drift-audit`; created thirteen execution-ready story notes under `.brain/planning/stories/`; and aligned each epic/spec with the locked order of work: hard packet budgets first, session packet reuse second, and derived capsules plus drift audit third.

- 2026-04-16: Tightened the automatic project-upgrade UX at bootstrap. Fresh `brain init`, `brain adopt`, and `brain context install` flows now initialize the repo-local project-migration ledger as current, so a brand-new Brain repo does not show `project_migrations: pending` in `brain doctor` before any unrelated preflight command has run. Also closed the release/install planning cleanup by marking the spec approved now that the migration lifecycle work is fully implemented.

- 2026-04-16: Updated the user-facing and maintainer-facing guidance for automatic project upgrades. `docs/usage.md`, `docs/skills.md`, `docs/project-workflows.md`, `.brain/context/workflows.md`, the maintainer refresh reference, and `skills/brain/SKILL.md` now explain that `brain update` refreshes the current repo's pending project migrations, older Brain repos migrate lazily on first later use, `brain doctor` reports project migration health, the explicit fallback is still `brain doctor --project .`, `brain context refresh --project .`, and `brain adopt --project .`, and migration changes should be validated from a branch-built binary with `go run . context migrate --project <repo>` before merge.

- 2026-04-15: Surfaced automatic project-migration health everywhere Brain already reports upgrade state. `brain update` now emits project migration status plus applied migration ids in human and JSON output for Brain repos, lazy preflight migration failures block work with remediation that points to `brain doctor`, `brain context refresh --project .`, and `brain adopt --project .`, and `brain doctor` now reports project migrations as `current`, `pending`, or `broken` using the same repo-local migration planner state.

- 2026-04-15: Wired project soft migrations into the actual Brain command lifecycle. `brain update` now runs project migrations for the current `--project` after binary install and skill refresh by invoking the freshly selected binary through a hidden `brain context migrate` command, `brain update --check` stays read-only, and normal app-backed commands now run one per-process project-repair preflight that repairs local Brain skills and applies pending project migrations lazily for older Brain repos while skipping bootstrap-only or mutation-free top-level commands.

- 2026-04-15: Implemented the first-wave automatic soft project migrations in `internal/projectcontext/`. Brain now has an idempotent project-migration runner that reuses the managed-context refresh path plus existing-agent integration sync, applies named migration ids into the new repo-local ledger, refreshes stale Brain-managed docs, migrates legacy agent wrapper blocks in place, leaves unmanaged agent files alone, and reports clean `unchanged` behavior on reruns once a repo is current.

- 2026-04-15: Added the first project soft-migration state model under `internal/projectcontext/`. Brain now has a repo-local migration ledger path at `.brain/state/project-migrations.json`, a named migration registry for first-wave soft upgrades, planner APIs that compare applied migration ids instead of raw Brain version strings, recoverable handling for missing or invalid migration state, and a guard that refuses to write migration state into repos that do not already use Brain.

- 2026-04-15: Extended the `release-install-and-update-flow` planning track so Brain upgrades will eventually own automatic soft project migrations too, not just binary and skill refresh. The current plan is to add a repo-local project migration ledger, reuse idempotent `context refresh` plus agent-integration sync primitives for first-wave migrations, run them automatically during `brain update` for the current `--project` and lazily on first Brain use in older repos, surface migration health in `brain doctor` and `brain update`, and update the Brain skill/docs alongside the implementation.

- 2026-04-15: Completed the `v4` context-compiler rollout slices for promotion gating, closeout suggestions, and compiler-era UX migration. Brain now classifies first-wave durable-memory candidates through `internal/promotion`, surfaces packet-backed promotion suggestions during blocked closeout, renders `brain distill --session` as a promotion-review note instead of a fixed target list, teaches `brain context compile` as the primary context surface, and refreshes generated repo guidance plus the Brain skill around promotion-aware closeout.

- 2026-04-15: Completed the `v3` context-compiler rollout slices for local packet telemetry, packet inspection surfaces, and conservative utility-aware ranking. Brain now records compile, expansion, verification, durable-update, and closeout events in session telemetry; exposes `brain context explain` and `brain context stats` for packet rationale and local signal/noise inspection; and uses repeated local expansions plus downstream outcomes to apply bounded utility boosts or penalties to future compiler note selection with explicit diagnostics.

- 2026-04-15: Completed the `v2` context-compiler rollout slices for boundary-aware selection and verification surfaces. Brain now derives compiler-facing boundary graphs with adjacency, responsibilities, and owned tests; `context live` and `context compile` use those boundaries for touched-boundary, nearby-test, and durable-note selection; compiled packets keep boundary-aware nearby-test relations plus explicit provenance; and live/compiled context now surface repo-derived verification recipes from policy, Makefile targets, package scripts, CI workflows, and bounded successful session commands with strong-or-suggested guidance.

- 2026-04-15: Added the first `v1` context-compiler surface with compiler-facing context item types, compact base-contract extraction, the new `brain context compile` command, summary-first packet output with anchors and provenance, `internal/taskcontext/` as the first compiler package, and active-session packet recording for compiled working sets.

- 2026-04-14: Bundled the Brain skill into the running binary, removed symlink mode from `brain skills`, added `.brain-skill-manifest.json` freshness tracking, taught `brain update` plus both install scripts to refresh existing Brain skill installs, and added lazy local skill auto-repair before app-backed Brain commands run.

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
- 2026-04-11: Removed the final legacy planning compatibility path from `.brain/project.yaml` so Brain now accepts only `planning_model: epic_spec_v1`, and rewrote the README flow so install is followed immediately by adding the Brain skill and then by the brainstorm -> epic -> story execution workflow.
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
- 2026-04-11: Fixed the session lock helper for Windows by treating `os.ErrPermission` on an already-existing lock directory as normal contention rather than a hard failure. This prevents concurrent `brain session run` tests from failing on Windows with `Access is denied` while preserving the filesystem lock behavior.
- 2026-04-11: Returned the local checkout to `main` after PR #3 merged and fast-forwarded the repo to `v0.1.5`, which includes the wrapper cleanup, Windows session-lock fix, and corrected reusable release input handling.
- 2026-04-11: Repositioned the product docs around Brain as durable local operating memory for AI coding agents. The README now leads with the continuity/reliability story, keeps install + skill install + brainstorm-to-execution prominent, treats lower token and tool cost as supporting proof, and replaces the bottom link dump with high-level overview sections that deep-link into usage, architecture, skills, and why.
- 2026-04-11: Tightened the Windows session-lock fix again after PR feedback. Lock acquisition now treats Windows `os.ErrPermission` during lock-directory races as retryable contention even if a follow-up stat misses the directory, which should stop intermittent `Access is denied` failures in concurrent session tests.
- 2026-04-11: Tightened the README positioning pass by adding an explicit audience section for heavy AI-agent users, a `Why Brain Exists` founder-pain section, and a cleaner deep-dive section structure so the README sells urgency before dropping into technical overviews.
- 2026-04-12: Returned the local checkout to `main` after PR #5 merged and fast-forwarded the repo to `v0.1.7`, which includes the README urgency follow-up and aligned supporting docs.

- 2026-04-13: Updated `skills/brain/SKILL.md` and refreshed the global Codex `brain` skill so it now teaches `brain distill`, `brain search --inject`, `brain context load --level`, and the session-finish distill recovery flow.

- 2026-04-13: Fixed the CLI test output normalizer so Windows no longer corrupts JSON escape sequences while replacing temp-root paths, which restores `TestCLIContextLoadLevels` and other JSON-based CLI tests on the Windows pipeline.

- 2026-04-13: Fixed the remaining Windows CLI skill-target assertions to use OS-native path joins in tests, so `TestCLISkillsCommands` now accepts Windows `\` paths without regressing Unix output expectations.
- 2026-04-14: Reworked project-context agent integrations so `brain context install` and `brain context refresh` no longer create agent-specific instruction files, `brain adopt` now appends or updates Brain-managed sections inside existing local agent files, `brain adopt --agent ...` is the only creation path for a missing local agent instruction file, and the generated Brain guidance no longer declares any AI file canonical.
- 2026-04-14: Refreshed the checked-in project context from the current branch so `AGENTS.md`, `docs/project-workflows.md`, and `.brain/context/workflows.md` now reflect the supplemental agent-integration model and the neutral Brain wording.
- 2026-04-14: Closed the follow-up review gaps in agent integration: Pi is now a first-class auto-detected agent target, unsupported `--agent` values fail fast instead of creating arbitrary directories, and legacy Brain wrapper files migrate in place to the new `agent-integration-*` block format without preserving stale canonical-language wrapper text.
