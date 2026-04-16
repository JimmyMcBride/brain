---
title: "Agent Workflow"
type: "reference"
created: "2026-04-11T00:00:00Z"
updated: "2026-04-11T00:00:00Z"
source: "migrated_project_memory"
---
# Agent Workflow

## Startup Checklist

1. Run `brain doctor`.
2. Read `AGENTS.md` and the linked `.brain/context/*.md` files relevant to the task.
3. Retrieve project-local memory with `brain find brain` or `brain search "brain <task>"`.
4. Check `git status --short --branch` before editing.
5. Start a session when the repo contract requires enforcement.

## Guide Selection

- Product overview and command surface: `README.md`, `docs/usage.md`
- Internal structure and boundaries: `docs/architecture.md`
- Skill and wrapper behavior: `docs/skills.md`, `skills/brain/SKILL.md`
- Root contract and repo-local context: `AGENTS.md`, `.brain/context/*`

## Durable Memory Rule

Update durable notes when command behavior, search/index semantics, context generation, session behavior, or release/update flows change. Prefer updating an existing note over producing noise.
