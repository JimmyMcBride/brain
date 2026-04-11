---
updated: "2026-04-11T15:28:38Z"
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

- 2026-04-11: Added the repo-owned `googleworkspace-cli` skill bundle, installed it to `~/.codex/skills/googleworkspace-cli`, and documented the one-line install path via `scripts/install.sh`.
- 2026-04-11: Hardened note updates to normalize full-note stdin/frontmatter safely and made `brain skills` install repo-owned skills as a bundle by default.
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
