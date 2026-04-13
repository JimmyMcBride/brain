---
title: "Maintainer Global Refresh"
type: "reference"
created: "2026-04-11T00:00:00Z"
updated: "2026-04-11T00:00:00Z"
---
# Maintainer Global Refresh

Use this when a real product change should also update your installed `brain` binary and the global Codex `brain` skill.

Default maintainer flow: work on a feature branch, open a PR, merge to `main`, wait for the automatic release, then refresh your local install.

## Sequence

1. Create or switch to a feature branch from `main`.
2. Finish the feature, bug fix, or doc change on that branch.
3. Record any durable note updates in repo memory.
4. Run required verification through `brain session run -- <command>`.
5. Commit the branch changes.
6. Open a PR into `main`.
   - Write the PR title and body in release-note language because GitHub release notes are generated from merged PR metadata.
   - Summarize shipped behavior in human-readable bullets, not just implementation steps or internal refactors.
7. Review and merge the PR.
8. Wait for the automatic stable release workflow to tag and publish the new version from that merge commit on `main`.
9. Refresh the installed binary and global Codex skill:

Unix shell:

```bash
./scripts/refresh-global-brain.sh
```

Windows PowerShell:

```powershell
.\scripts\refresh-global-brain.ps1
```

10. Verify:
   - Unix: `~/.local/bin/brain version` shows the pushed commit
   - Windows: `%LocalAppData%\Programs\brain\brain.exe version` shows the pushed commit
   - the installed global Codex `brain` skill matches `skills/brain/`

## Defaults

- This flow refreshes only the global Codex `brain` skill.
- It does not commit, push, or edit repo-tracked files.
- Treat direct pushes to `main` as the exception, not the default. PR merge is the normal release boundary.
- Treat release-note-friendly PR copy as part of the definition of done for every PR.
- Do not create a follow-up repo-memory commit just because you refreshed the global binary or skill. Otherwise the installed binary immediately lags `HEAD` again.
