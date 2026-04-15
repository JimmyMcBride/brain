# Skills

`brain` installs one skill bundle: the Brain skill itself.

The point of the Brain skill is simple: it teaches the agent how to operate against a Brain-managed repo immediately, instead of expecting the agent to infer the workflow from scattered files and conventions.

## Brain Skill Bundle

- `skills/brain/SKILL.md`
- `skills/brain/agents/openai.yaml`
- `skills/brain/agents/openclaw.yaml`

The Brain skill is the generic fallback for project-local Brain workflows, memory, compiled task context, and sessions.

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
brain skills install --scope local --agent openclaw --project .
brain skills install --scope local --agent copilot --project .
brain skills install --scope local --agent pi --project .
```

`brain` always installs copied skill directories. It never symlinks the Brain skill.

`brain skills install` and `brain skills targets` work from any directory because the Brain skill is bundled into the running binary instead of being resolved from the current working tree.

Default roots:

- Codex global: `~/.codex/skills/`
- Codex local: `.codex/skills/`
- Claude global: `~/.claude/skills/`
- Copilot global: `~/.copilot/skills/`
- Pi global: `~/.pi/agent/skills/`
- OpenClaw global: `~/.openclaw/skills/`
- OpenClaw local: `.openclaw/skills/`
- Copilot local: `.github/skills/`
- Pi local: `.pi/skills/`

## Repo Maintenance

Installed skills include a generated `.brain-skill-manifest.json` file beside `SKILL.md`. Brain uses that manifest to detect stale or legacy installs and repair local project skills before work begins.

When a repo change updates Brain's command surface or agent-facing workflow guidance, update `skills/brain/SKILL.md` in the same branch, validate the bundled skill with the current branch binary, and reinstall the local Brain skill for Codex and OpenClaw before closing the work:

```bash
go run . skills install --scope local --agent codex --agent openclaw --project .
```

Then reinstall or refresh with the installed binary:

```bash
brain skills install --scope local --agent codex --agent openclaw --project .
```

## Relationship To Project Context

The skill is the generic fallback.  
Repo-local context is the project-specific Brain surface.

Expected order:

1. read repo `AGENTS.md`
2. read `.brain/policy.yaml`
3. read the relevant `.brain/context/*.md` files
4. fall back to the generic Brain skill only when repo-local context is absent or insufficient

## Agent Integrations

`brain context install` and `brain context refresh` do not create missing agent-specific instruction files.

`brain adopt` scans for existing local agent files such as `.codex/AGENTS.md`, `.claude/CLAUDE.md`, or `.pi/AGENTS.md` and appends or updates a Brain-managed section that points agents to `.brain/` and the Brain workflow.

`brain adopt --agent ...` is the explicit path that may create a missing local agent instruction file. Unsupported agent names are rejected instead of creating arbitrary directories.

## Sessions And Verification

When a repo uses sessions, the skill should steer agents toward:

- `brain session start`
- `brain session validate`
- `brain session run -- <command>`
- `brain session finish`

If finish blocks, the skill should steer agents toward the closeout promotion suggestions first and then to `brain distill --session` for the full promotion review note.

That keeps verification and durable memory updates visible and enforceable.
