# Skills

`brain` ships canonical skill bundles for agent runtimes.

## Bundle Layout

- `skills/brain/SKILL.md`
- `skills/brain/agents/openai.yaml`
- `skills/brain/agents/openclaw.yaml`
- `skills/googleworkspace-cli/SKILL.md`
- `skills/googleworkspace-cli/agents/openai.yaml`
- `skills/googleworkspace-cli/references/*.md`
- `skills/wrappers/*.md`

## Included Skills

- `brain`
  - generic fallback for project-local Brain workflows, memory, planning, brainstorming, context, and sessions
- `googleworkspace-cli`
  - generic `gws` skill for Google Workspace terminal work across Drive, Gmail, Calendar, Sheets, Docs, Chat, and related APIs

## Install Targets

Preview install targets:

```bash
brain skills targets --scope both --agent codex --agent claude --project .
brain skills targets --scope global --agent codex --skill brain
```

Install globally:

```bash
brain skills install --scope global --agent codex
brain skills install --scope global --agent codex --skill googleworkspace-cli
```

Install into a project:

```bash
brain skills install --scope local --agent codex --project .
```

Use `--mode copy` when the target runtime does not support symlinked skill directories well. OpenClaw should generally use copy mode.
When no `--skill` flag is provided, `brain` installs all repo-owned skills discovered under `./skills`.

Global Codex install targets land under `~/.codex/skills/`.

## Relationship To Project Context

The skill is the generic fallback.  
Repo-local context is the project-specific contract.

Expected order:

1. read repo `AGENTS.md`
2. read `.brain/policy.yaml`
3. read the relevant `.brain/context/*.md` files
4. fall back to the generic Brain skill only when repo-local context is absent or insufficient

## Wrappers

Agent-specific wrappers such as `.codex/AGENTS.md` or `.claude/CLAUDE.md` are intentionally thin. They should point back to the root contract instead of duplicating policy.

## Sessions And Verification

When a repo uses sessions, the skill should steer agents toward:

- `brain session start`
- `brain session validate`
- `brain session run -- <command>`
- `brain session finish`

That keeps verification and durable memory updates visible and enforceable.
