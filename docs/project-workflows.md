# Project Workflows

<!-- brain:begin project-doc-workflows -->
Use this file for agent operating workflow inside the repo.

## Startup

1. If no validated session is active, run `brain session start --task "<task>"`.
2. If a session already exists, run `brain session validate`.
3. Read `AGENTS.md`, `.brain/policy.yaml`, and the linked context files needed for the task.
4. If project memory matters, run `brain find brain` or `brain search "brain <task>"`.

## During Work

- Keep durable discoveries, decisions, and risks in AGENTS.md, /docs, or .brain notes.
- Update existing durable notes instead of duplicating context.
- Run required verification commands through `brain session run -- <command>`.
- Re-read context before large changes if the task shifts.

## Close-Out

- Refresh or update durable notes for meaningful behavior, config, or architecture changes.
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
- [.brain/resources/changes/project-scoped-planning-and-brainstorming.md](../.brain/resources/changes/project-scoped-planning-and-brainstorming.md)
