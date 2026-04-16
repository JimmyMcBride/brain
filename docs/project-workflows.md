---
updated: "2026-04-16T02:42:19Z"
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
- If `brain session finish` blocks, inspect the promotion suggestions or run `brain distill --session` to review promotable updates before forcing closeout.
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

Maintainer release flow:

- Do product work on a branch.
- Verify through `brain session run`.
- Commit the branch.
- Open and merge a PR to `main`.
- Let the merge trigger the automatic release.
- Refresh the installed binary and global Codex Brain skill only after that release exists.
