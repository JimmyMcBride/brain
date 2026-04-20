---
updated: "2026-04-20T05:36:15Z"
---
# Project Workflows

<!-- brain:begin project-doc-workflows -->
Use this file for agent operating workflow inside the repo.

## Startup

1. If no validated session is active, run `brain session start --task "<task>"`.
2. If a session already exists, run `brain session validate`.
3. Read `AGENTS.md`, `.brain/policy.yaml`, and the linked context files needed for the task.
4. Run `brain context compile --task "<task>"` for the smallest justified working set.
5. If project memory still matters, run `brain find brain` or `brain search "brain <task>"`.

## During Work

- Keep durable discoveries, decisions, and risks in AGENTS.md, /docs, or .brain notes.
- Update existing durable notes instead of duplicating context.
- Run required verification commands through `brain session run -- <command>`.
- If you change Brain command behavior or agent-facing workflow guidance, update `skills/brain/SKILL.md` in the same branch.
- Re-read context before large changes if the task shifts.

## Ticket Loop

1. Start one task or ticket at a time and keep the scope narrow.
2. Implement the task, then run focused tests for the touched packages.
3. Run the required full checks through `brain session run -- go test ./...` and `brain session run -- go build ./...`.
4. Review the diff against the task goal and user-facing behavior.
5. If review finds issues, patch the work and repeat the test and review steps.
6. When the task is clean, commit it, push it, and only then move to the next task.

## Close-Out

- Refresh or update durable notes for meaningful behavior, config, or architecture changes.
- If `brain session finish` blocks, inspect the promotion suggestions first; run `brain distill --session --dry-run` only when you need the full review without creating a proposal note.
- Before switching away from a working branch or back to `develop`, run `git status --short` and resolve repo-owned leftovers. If `.brain/resources/changes/*`, `.brain/`, `docs/`, or contract files belong to the task, keep them in the same branch/PR; otherwise review and intentionally remove them instead of carrying them onto `develop`, `release/*`, or `main`.
- If `skills/brain/` changed, reinstall the local Brain skill for Codex and OpenClaw with `brain skills install --scope local --agent codex --agent openclaw --project .`.
- When opening a PR, make the title and body release-note friendly because GitHub release notes are generated from merged PR metadata.
- Summarize shipped behavior in the PR, not just implementation steps, so future changelogs stay human-readable.
- Finish with `brain session finish`.
- If you must bypass enforcement, use `brain session finish --force --reason "..."` so the override is recorded.
<!-- brain:end project-doc-workflows -->

## Local Notes

Historical workflow references:

- [.brain/resources/references/agent-workflow.md](../.brain/resources/references/agent-workflow.md)
- [.brain/resources/references/testing-and-operations.md](../.brain/resources/references/testing-and-operations.md)

Release/history references:

- [.brain/resources/changes/project-context-bundles-and-agent-contracts.md](../.brain/resources/changes/project-context-bundles-and-agent-contracts.md)
- [.brain/resources/changes/session-enforcement-and-policy-engine.md](../.brain/resources/changes/session-enforcement-and-policy-engine.md)
- [.brain/resources/references/maintainer-global-refresh.md](../.brain/resources/references/maintainer-global-refresh.md)

Maintainer gitflow:

- Treat `develop` as the active long-lived integration branch.
- Treat `release/vX.Y.Z` as the protected release stabilization branch cut from `develop`.
- Treat `main` as the protected production branch; merging to `main` is the publish event.
- Never push directly to `develop`, `release/*`, or `main`; use PRs for all protected-branch changes.
- Start routine feature and bug-fix work from latest `develop` and open PRs back into `develop`.
- When a release needs a fix, land it in `develop` first and then cherry-pick the exact commit into the active `release/vX.Y.Z` branch.
- For urgent production-only fixes, branch from the active `release/vX.Y.Z` branch or from `main`, then make sure the equivalent fix lands back in `develop`.
- Preserve release branches as historical snapshots.
- After every PR merge into `develop`, fetch latest remote state, check out updated `origin/develop`, and refresh Brain context. Refresh `.plan/` context too if a repo-local plan workspace exists.
- Before returning to `develop`, resolve repo-owned `.brain/resources/changes/*`, `.brain/`, `docs/`, and contract-file leftovers on the feature branch. Merge them in the active PR if they belong to the task; otherwise review and intentionally remove them.
- Cut `release/vX.Y.Z` from current `develop` when preparing an official release, then merge that release PR into `main` only when ready to publish.
- Refresh the installed binary and global Codex Brain skill only after the release merge has published from `main`.
