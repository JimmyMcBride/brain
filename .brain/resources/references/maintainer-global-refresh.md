---
created: "2026-04-11T00:00:00Z"
title: Maintainer Global Refresh
type: reference
updated: "2026-04-20T05:36:34Z"
---
# Maintainer Global Refresh

Use this when a real product change should also update your installed `brain` binary and the global Codex `brain` skill.

Default maintainer flow: work on a feature branch from the current `develop` line, merge routine work into `develop`, cut `release/vX.Y.Z` from `develop` for official releases, merge the release PR into `main`, wait for the automatic publish, then refresh your local install.

## Sequence

1. Create or switch to a feature branch from latest `develop`.
2. Finish the feature, bug fix, or doc change on that branch.
3. Record any durable note updates in repo memory.
4. Run required verification through `brain session run -- <command>`.
5. If the branch changed automatic project-upgrade behavior, validate it from the branch-built binary against a representative older Brain repo before you commit or merge:

```bash
go run . context migrate --project ../older-brain-repo
```

6. If the branch changed `skills/brain/` or other agent-facing workflow guidance, validate the bundled skill from the branch-built binary too:

```bash
go run . skills install --scope local --agent codex --agent openclaw --project .
```

7. Commit the branch changes.
   - Before you commit or switch away from the branch, run `git status --short` and resolve repo-owned leftovers. If `.brain/resources/changes/*`, `.brain/`, `docs/`, or contract files belong to the task, keep them in this branch and PR; otherwise review and intentionally remove them.
8. Open a PR into `develop`.
   - Write the PR title and body in release-note language because the release workflow publishes the merged release PR's `## Release Notes` section as the release changelog, with `## User-Facing Impact` or `## Summary` as fallback.
   - Fill `## Release Notes` with high-signal, human-readable bullets, not implementation steps or internal refactors.
9. Review and merge the PR into `develop`.
10. Immediately fetch latest remote state, check out updated `origin/develop`, and refresh Brain project context so local reasoning matches newest `develop`. If a repo-local `.plan/` workspace exists, refresh that context too.
11. When an official release is ready, create `release/vX.Y.Z` from current `develop`.
12. If release stabilization needs a fix, land the fix in `develop` first and then cherry-pick the exact commit into the active `release/vX.Y.Z` branch.
13. Open a PR from `release/vX.Y.Z` into `main`.
14. Review and merge the release PR into `main` when ready to publish.
15. Wait for the automatic stable release workflow to tag and publish the new version from that merge commit on `main`.
16. Refresh the installed binary and global Codex skill:

Unix shell:

```bash
./scripts/refresh-global-brain.sh
```

Windows PowerShell:

```powershell
.\scripts\refresh-global-brain.ps1
```

17. Verify:
   - Unix: `~/.local/bin/brain version` shows the pushed commit
   - Windows: `%LocalAppData%\Programs\brain\brain.exe version` shows the pushed commit
   - the installed global Codex `brain` skill matches `skills/brain/`

## Defaults

- `develop`, `release/*`, and `main` are protected branches.
- Never push directly to `develop`, `release/*`, or `main`.
- Never delete `develop`, `release/*`, or `main`.
- Hotfixes may branch from active `release/vX.Y.Z` or from `main`, whichever best matches production, but the equivalent fix must always land back in `develop`.
- Preserve release branches as historical snapshots for regression and release inspection.
- Do not return to `develop`, `release/*`, or `main` with repo-owned proposal or context files still hanging out in the worktree.
- Treat release-note-friendly PR copy as part of the definition of done for every PR.
- Do not create a follow-up repo-memory commit just because you refreshed the global binary or skill. Otherwise the installed binary immediately lags `HEAD` again.
