# Skills

`brain` installs one skill bundle: the Brain skill itself.

## Brain Skill Bundle

- `skills/brain/SKILL.md`
- `skills/brain/agents/openai.yaml`
- `skills/brain/agents/openclaw.yaml`

The Brain skill is the generic fallback for project-local Brain workflows, memory, epic/spec/story planning, brainstorming, context, and sessions.

## Install Targets

Preview install targets:

```bash
brain skills targets --scope both --agent codex --agent claude --agent copilot --agent pi --project .
```

Install globally:

```bash
brain skills install --scope global --agent codex
brain skills install --scope global --agent claude
brain skills install --scope global --agent copilot
brain skills install --scope global --agent pi
```

Install into a project:

```bash
brain skills install --scope local --agent codex --project .
brain skills install --scope local --agent copilot --project .
brain skills install --scope local --agent pi --project .
```

Use `--mode copy` when the target runtime does not support symlinked skill directories well. OpenClaw should generally use copy mode.

Default roots:

- Codex global: `~/.codex/skills/`
- Claude global: `~/.claude/skills/`
- Copilot global: `~/.copilot/skills/`
- Pi global: `~/.pi/agent/skills/`
- Copilot local: `.github/skills/`
- Pi local: `.pi/skills/`

## Relationship To Project Context

The skill is the generic fallback.  
Repo-local context is the project-specific contract.

Expected order:

1. read repo `AGENTS.md`
2. read `.brain/policy.yaml`
3. read the relevant `.brain/context/*.md` files
4. fall back to the generic Brain skill only when repo-local context is absent or insufficient

## Wrappers

Agent-specific wrappers such as `.codex/AGENTS.md` or `.claude/CLAUDE.md` are optional and are generated only when you explicitly request them with `brain context install --agent ...` or `brain context refresh --agent ...`. They should stay thin and point back to the root contract instead of duplicating policy.

## Sessions And Verification

When a repo uses sessions, the skill should steer agents toward:

- `brain session start`
- `brain session validate`
- `brain session run -- <command>`
- `brain session finish`

That keeps verification and durable memory updates visible and enforceable.
