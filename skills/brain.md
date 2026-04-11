---
name: brain
description: Use this skill when the task depends on a project-local Brain workspace managed by the `brain` CLI.
user-invocable: true
args:
  - name: task
    description: The Brain or project-memory task to perform.
    required: false
---

# Brain Skill

Use this skill when project memory, retrieval, planning, brainstorming, or agent context lives inside the current repo.

## Operating Model

- Treat the repo-local Brain workspace as authoritative.
- Keep durable knowledge in `AGENTS.md`, `docs/`, and `.brain/`.
- Prefer Brain CLI commands over raw file churn when Brain already owns the note.
- Use `brain search` for ranked retrieval and `brain find` for direct matching.

## High-Value Commands

- `brain init`
- `brain doctor`
- `brain read`
- `brain edit`
- `brain find`
- `brain search`
- `brain brainstorm`
- `brain plan`
- `brain context install|refresh`
- `brain session start|validate|run|finish`
- `brain history`
- `brain undo`

## Suggested Workflow

1. Read repo `AGENTS.md` and `.brain/context/*`.
2. Retrieve relevant context with `brain find` or `brain search`.
3. Make durable note updates with `brain edit` or Brain-managed generators.
4. Record required verification through `brain session run -- ...` when sessions are active.
