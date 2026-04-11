---
title: "Skills And Context Engineering"
type: "reference"
created: "2026-04-11T00:00:00Z"
updated: "2026-04-11T00:00:00Z"
source: "migrated_project_memory"
---
# Skills And Context Engineering

## Skill System

The canonical skill bundle lives under `skills/brain/`.

- `skills/brain/SKILL.md`
- `skills/brain/agents/openai.yaml`
- `skills/brain/agents/openclaw.yaml`

Installation is handled by `brain skills targets` and `brain skills install`.

## Project Context System

Repo-local context is generated through:

- `brain context install --project .`
- `brain context refresh --project .`

Managed files include:

- `AGENTS.md`
- `.brain/context/*`
- `.brain/policy.yaml`
- thin agent wrappers such as `.codex/AGENTS.md` and `.claude/CLAUDE.md`

## Managed-Block Rules

- Brain owns only marked sections or whole generated files.
- Refreshes must preserve user-authored local notes outside managed blocks.
- Wrappers should delegate back to the root contract instead of duplicating policy.
