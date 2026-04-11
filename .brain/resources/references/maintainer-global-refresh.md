---
title: "Maintainer Global Refresh"
type: "reference"
created: "2026-04-11T00:00:00Z"
updated: "2026-04-11T00:00:00Z"
---
# Maintainer Global Refresh

Use this when a real product change should also update your installed `brain` binary and the global Codex `brain` skill.

## Sequence

1. Finish the feature, bug fix, or doc change.
2. Record any durable note updates in repo memory.
3. Run required verification through `brain session run -- <command>`.
4. Commit the repo changes.
5. Push `main`.
6. Wait for the automatic stable release workflow to tag and publish the new version from that pushed commit.
7. Refresh the installed binary and global Codex skill:

Unix shell:

```bash
./scripts/refresh-global-brain.sh
```

Windows PowerShell:

```powershell
.\scripts\refresh-global-brain.ps1
```

8. Verify:
   - Unix: `~/.local/bin/brain version` shows the pushed commit
   - Windows: `%LocalAppData%\Programs\brain\brain.exe version` shows the pushed commit
   - the installed global Codex `brain` skill matches `skills/brain/`

## Defaults

- This flow refreshes only the global Codex `brain` skill.
- It does not commit, push, or edit repo-tracked files.
- Do not create a follow-up repo-memory commit just because you refreshed the global binary or skill. Otherwise the installed binary immediately lags `HEAD` again.
