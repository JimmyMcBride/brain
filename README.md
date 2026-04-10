# brain

`brain` is a production-oriented Go CLI for local markdown knowledge work:

- Linux-first, Arch-friendly
- Obsidian-compatible markdown vault as source of truth
- PARA at the top level
- Hybrid retrieval with SQLite FTS5 plus embeddings
- Agent-friendly commands
- Project-local context bundles for coding agents
- Backups, history, undo, and diffable organize workflows

## Install

### Build from source

```bash
git clone https://github.com/JimmyMcBride/brain.git
cd brain
go build -o brain .
sudo install -m 0755 brain /usr/local/bin/brain
```

### Go install

```bash
go install .
```

## Quick start

```bash
brain init
brain doctor
brain add "AI Agent Workflow" --section Projects --type project
brain capture "Interesting idea" --body "A short note about retrieval and agents."
brain daily
brain reindex
brain search "retrieval agents"
```

## Config

Config lives at `~/.config/brain/config.yaml` by default.

Supported fields:

- `vault_path`
- `data_path`
- `embedding_provider`
- `embedding_model`
- `output_mode`

Environment overrides:

- `BRAIN_VAULT_PATH`
- `BRAIN_DATA_PATH`
- `BRAIN_EMBEDDING_PROVIDER`
- `BRAIN_EMBEDDING_MODEL`
- `BRAIN_OUTPUT_MODE`

## Command examples

```bash
brain add "Client migration" --section Projects --type project
brain read Projects/client-migration.md
brain edit Projects/client-migration.md --set status=active
brain find migration
brain search "vendor rollout plan"
brain move Projects/client-migration.md Archives/
brain history
brain undo
```

## Content workflow

```bash
brain content seed Projects/client-migration.md
brain content gather Projects/client-migration.md -n 5
brain content outline Projects/client-migration.md -n 5
brain content publish Projects/client-migration.md --channel blog --repurpose thread
```

## Skills

```bash
brain skills install --scope global --agent codex
brain skills install --scope local --agent codex --project .
brain skills install --scope both --agent codex --agent claude --project .
brain skills install --scope global --agent openclaw
brain skills install --skill-root /path/to/custom/skills --mode copy
brain context install --project . --agent codex --agent openclaw
brain context refresh --project .
```

OpenClaw installs are copied into `~/.openclaw/skills/brain` because OpenClaw's managed skill loader does not currently detect symlinked skill directories.

`brain context install` creates a root `AGENTS.md`, a modular `.brain/context` bundle, and thin agent-specific wrappers so coding agents can follow a consistent project contract.

## Example vault structure

```text
vault/
  Projects/
    ai-agent-workflow.md
  Areas/
    Daily/
      2026/
        2026-04-09.md
  Resources/
    Captures/
      2026/
        04/
          interesting-idea.md
    Content/
      Outlines/
        ai-agent-workflow-outline.md
  Archives/
```

## Example search queries

```bash
brain search "retrieval agents"
brain search "weekly review"
brain search "publishing workflow"
brain find --type project
```

## Linux setup

1. Install Go and a C toolchain only if you want to build other CGO-based tooling; `brain` itself uses a pure-Go SQLite driver.
2. Build and place `brain` on your `PATH`.
3. Run `brain init` and confirm with `brain doctor`.
4. Point Obsidian at the configured `vault_path`.
5. Run `brain reindex` after meaningful note imports or edits.

## More docs

- [Architecture](docs/architecture.md)
- [Usage](docs/usage.md)
- [Skills](docs/skills.md)
- [Why](docs/why.md)
