# Usage

`brain` is operated per project. Use `--project` when you are acting on a repo other than the current directory.

The default config uses `embedding_provider: localhash` and `embedding_model: hash-v1`.

## Install

For the standard end-user install path on Unix:

```bash
curl -fsSL https://raw.githubusercontent.com/JimmyMcBride/brain/main/scripts/install.sh | sh
```

For Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/JimmyMcBride/brain/main/scripts/install.ps1 | iex
```

That installs the latest tagged release with checksum verification into:

- Unix: `~/.local/bin/brain`
- Windows: `%LocalAppData%\Programs\brain\brain.exe`

Stable GitHub releases are published from `main`. Prefer PR merges as the normal path into `main`, then install and update from the latest release published there.

If no release has been published yet, the installer falls back to downloading the current `main` source tarball and building it locally with Go.

## Bootstrap A Project

For a new or mostly empty repo:

```bash
brain init --project .
brain doctor --project .
brain context install --project .
brain plan init --project .
```

For an existing repo that already has docs or an unmanaged `AGENTS.md`:

```bash
brain adopt --project .
brain doctor --project .
brain plan init --project .
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

`brain init` is the clean bootstrap path.  
`brain adopt` is the existing-repo path: it creates the local Brain workspace, adopts Brain-owned docs into the managed-block model, and preserves previous content under `Local Notes`.

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
brain search --project . status
brain search --project . --explain "Supabase auth"
```

`find` is path/title/type/content matching.  
`search` uses the local SQLite index plus the configured embedding provider over project-managed markdown. With the default `localhash` provider, the result is strong local lexical search plus lightweight semantic hinting rather than a high-quality hosted semantic model. The index lives in `.brain/state/brain.sqlite3`, tracks its own freshness, rebuilds automatically when it is missing or stale, and `brain doctor` plus `brain search status` both show the active provider/model. Use `--explain` to inspect lexical and semantic contributions.

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
brain plan init --project .
```

Brainstorm -> epic -> spec -> stories:

```bash
brain brainstorm start --project . "Forms API Hardening"
brain plan epic promote --project . forms-api-hardening
brain plan spec status --project . forms-api-hardening --set approved
brain plan story create --project . forms-api-hardening "Validate API keys" \
  --body "Harden external submissions before broader rollout." \
  --criteria "Reject malformed bearer tokens" \
  --criteria "Return stable 401 responses" \
  --resource "[[.brain/resources/changes/forms-api-rollout.md]]"
brain plan story update --project . validate-api-keys --status in_progress
brain plan status --project .
```

Planning is intentionally opinionated:

- brainstorms capture the high-level what and why
- epics capture the feature or initiative
- each epic gets one canonical spec
- stories are created only after the spec is approved

## Context Management

Install or refresh project context:

```bash
brain context install --project . --agent codex --agent claude
brain context refresh --project .
brain context refresh --project . --dry-run
```

Use `--force` when adopting an existing unmanaged `AGENTS.md` or docs file into the managed-block model.

If you want the full existing-repo bootstrap instead of just context takeover, use:

```bash
brain adopt --project .
```

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
brain skills install --scope global --agent claude
brain skills install --scope global --agent copilot
brain skills install --scope local --agent copilot --project .
brain skills install --scope global --agent pi
brain skills install --scope local --agent pi --project .
brain skills install --scope global --agent openclaw --mode copy
```

`brain skills install` always installs the Brain skill. Use `--scope global` to add it to your machine and `--scope local --project .` to add it to the current project.

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

On Windows, `brain update` uses the same release assets and default install target as `scripts/install.ps1`.

By default, `brain update` tracks the latest stable GitHub release published from `main`.
