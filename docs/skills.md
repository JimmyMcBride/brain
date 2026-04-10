# Skills

`brain` ships with a canonical skill bundle:

- `skills/brain/SKILL.md`
- `skills/brain/agents/openai.yaml`
- `skills/brain/agents/openclaw.yaml`

## Install

```bash
brain skills install --scope global --agent codex
```

This installs:

- `brain/SKILL.md`
- `brain/agents/openai.yaml`

into the target skill roots.

Examples:

```bash
brain skills targets --scope both --agent codex --agent claude --project .
brain skills install --scope local --agent codex --project .
brain skills install --scope both --agent codex --agent zed --project .
brain skills install --scope global --agent openclaw
brain skills install --skill-root /path/to/custom/skills --mode copy
```

Known agents use conventional roots:

- global: `~/.<agent>/skills`
- local: `<project>/.<agent>/skills`

Use `--skill-root` for nonstandard tools. Use `--mode copy` when symlinks are undesirable. Symlinks are preferable during local development because changes in the repo propagate immediately to the installed skill files.

OpenClaw note: OpenClaw's managed skill loader expects a real directory under `~/.openclaw/skills`, so `brain skills install --agent openclaw` uses copy mode even if `--mode symlink` is requested.

## Project Context

Use the context commands when you want a repository to carry its own agent contract instead of relying only on a global skill:

```bash
brain context install --project . --agent codex --agent openclaw
brain context refresh --project .
```

This creates:

- `AGENTS.md`
- `.brain/context/overview.md`
- `.brain/context/architecture.md`
- `.brain/context/standards.md`
- `.brain/context/workflows.md`
- `.brain/context/memory-policy.md`
- `.brain/context/current-state.md`

If an agent wrapper is requested, `brain` also generates a thin wrapper such as `.codex/AGENTS.md` or `.claude/CLAUDE.md` that points back to the root project contract.
