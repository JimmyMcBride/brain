# Skills

`brain` ships a canonical skill bundle for agent runtimes.

## Bundle Layout

- `skills/brain/SKILL.md`
- `skills/brain/agents/openai.yaml`
- `skills/brain/agents/openclaw.yaml`
- `skills/wrappers/*.md`

## Install Targets

Preview install targets:

```bash
brain skills targets --scope both --agent codex --agent claude --project .
```

Install globally:

```bash
brain skills install --scope global --agent codex
```

Install into a project:

```bash
brain skills install --scope local --agent codex --project .
```

Use `--mode copy` when the target runtime does not support symlinked skill directories well. OpenClaw should generally use copy mode.

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
