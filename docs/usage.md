# Usage

`brain` is operated per project. Use `--project` when you are acting on a repo other than the current directory.

## Install

For the standard end-user install path:

```bash
curl -fsSL https://raw.githubusercontent.com/JimmyMcBride/brain/main/scripts/install.sh | sh
```

That installs the latest tagged release into `~/.local/bin/brain` with checksum verification.

If no release has been published yet, the installer falls back to downloading the current `main` source tarball and building it locally with Go.

## Bootstrap A Project

```bash
brain init --project .
brain doctor --project .
brain context install --project .
brain plan init --project . --paradigm epics
```

This creates:

- `AGENTS.md`
- `docs/project-overview.md`
- `docs/project-architecture.md`
- `docs/project-workflows.md`
- `.brain/context/*`
- `.brain/policy.yaml`
- `.brain/project.yaml`
- `.brain/state/brain.sqlite3`

## Read And Update Notes

```bash
brain read --project . AGENTS.md
brain read --project . docs/project-overview.md
brain edit --project . docs/project-overview.md -b "# Project Overview\n\nUpdated body."
brain edit --project . AGENTS.md --editor nvim
```

## Retrieve Context

```bash
brain find --project . auth
brain search --project . "Supabase auth"
```

`find` is path/title/type/content matching.  
`search` uses the local SQLite index and embeddings over project-managed markdown.

## Brainstorming

```bash
brain brainstorm start --project . "Event follow-up ideas"
brain read --project . .brain/brainstorms/event-follow-up-ideas.md
brain search --project . "follow-up"
```

Brainstorms live in `.brain/brainstorms/`.

## Planning

Initialize once:

```bash
brain plan init --project . --paradigm epics
```

Create containers and items:

```bash
brain plan group create --project . "Forms API Hardening"
brain plan item create --project . "Validate API keys" --group "Forms API Hardening" \
  --body "Harden external submissions before broader rollout." \
  --criteria "Reject malformed bearer tokens" \
  --criteria "Return stable 401 responses" \
  --resource "[[.brain/resources/changes/forms-api-rollout.md]]"
brain plan item update --project . validate-api-keys --status in_progress
brain plan status --project .
```

## Context Management

Install or refresh project context:

```bash
brain context install --project . --agent codex --agent claude
brain context refresh --project .
brain context refresh --project . --dry-run
```

Use `--force` when adopting an existing unmanaged `AGENTS.md` or docs file into the managed-block model.

## Sessions

Use sessions when the repo should require explicit verification and durable note updates.

```bash
brain session start --project . --task "tighten auth flow"
brain session validate --project .
brain session run --project . -- go test ./...
brain session run --project . -- go build ./...
brain session finish --project . --summary "auth flow tightened"
```

## Skills

```bash
brain skills targets --scope both --agent codex --project .
brain skills install --scope global --agent codex
brain skills install --scope local --agent codex --project .
brain skills install --scope global --agent openclaw --mode copy
```

## History And Undo

```bash
brain history --project .
brain undo --project .
```

These operate on Brain-managed note changes recorded in the local history log.

## Version And Update

```bash
brain version
brain update --check
brain update
```
