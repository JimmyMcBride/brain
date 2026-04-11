# 🧠 brain

`brain` is a local-first CLI that gives each software project its own markdown-based operating layer.

It keeps human docs at the repo root, keeps machine-managed state under `.brain/`, builds a local SQLite search index, and provides explicit workflows for planning, brainstorming, context, history, and session enforcement.

## What It Does

- initializes a project-local Brain workspace in any repo or folder
- keeps durable project knowledge in plain markdown
- indexes `AGENTS.md`, `docs/**/*.md`, and `.brain/**/*.md` with SQLite FTS plus an embedding provider
- provides project-scoped planning and brainstorming commands
- generates deterministic agent context and optional wrappers
- tracks note history and supports undo
- enforces repo workflows through session policy

## Mental Model

`brain` gives each project its own Brain: a clear markdown layer for docs, context, planning, brainstorming, and local search.

```text
my-project/
  AGENTS.md
  docs/
  .brain/
    context/
    brainstorms/
    planning/
    resources/
    state/
```

- `AGENTS.md` is the root contract for humans and agents.
- `docs/` is the human-readable project documentation layer.
- `.brain/context/` is the generated modular context bundle.
- `.brain/planning/` holds epics, specs, and stories.
- `.brain/brainstorms/` holds project-local ideation notes.
- `.brain/resources/` holds durable references, captures, and change history.
- `.brain/state/` holds SQLite, history logs, backups, and other local state.

## Install

### One-line install

Unix shell:

```bash
curl -fsSL https://raw.githubusercontent.com/JimmyMcBride/brain/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/JimmyMcBride/brain/main/scripts/install.ps1 | iex
```

These installers verify published checksums, support `linux`, `darwin`, and `windows` on `amd64` and `arm64`, and install by default to:

- Unix: `~/.local/bin/brain`
- Windows: `%LocalAppData%\Programs\brain\brain.exe`

Pushes to `main` publish stable GitHub releases automatically, so install normally targets the latest release from `main`.

If no GitHub release exists yet, the same command falls back to downloading the current `main` source archive from GitHub and building it locally with Go.

Optional overrides:

Unix shell:

```bash
curl -fsSL https://raw.githubusercontent.com/JimmyMcBride/brain/main/scripts/install.sh | \
  BRAIN_VERSION=v0.1.0 BRAIN_INSTALL_DIR="$HOME/.local/bin" sh
```

Windows PowerShell:

```powershell
$env:BRAIN_VERSION = "v0.1.0"
$env:BRAIN_INSTALL_DIR = "$env:LOCALAPPDATA\Programs\brain"
irm https://raw.githubusercontent.com/JimmyMcBride/brain/main/scripts/install.ps1 | iex
```

### Build from source

Unix shell:

```bash
git clone https://github.com/JimmyMcBride/brain.git
cd brain
go build -o brain .
install -Dm0755 brain ~/.local/bin/brain
```

Windows PowerShell:

```powershell
git clone https://github.com/JimmyMcBride/brain.git
cd brain
go build -o brain.exe .
New-Item -ItemType Directory -Force -Path "$env:LOCALAPPDATA\Programs\brain" | Out-Null
Copy-Item .\brain.exe "$env:LOCALAPPDATA\Programs\brain\brain.exe" -Force
```

### Go run during development

```bash
go run . --help
```

Use `go run .` when working on the CLI itself before replacing the installed binary.

## Install The Brain Skill

Add the Brain skill to your machine:

```bash
brain skills install --scope global --agent codex
brain skills install --scope global --agent claude
brain skills install --scope global --agent copilot
brain skills install --scope global --agent pi
```

Add the Brain skill to the current project:

```bash
brain skills install --scope local --agent codex --project .
brain skills install --scope local --agent copilot --project .
brain skills install --scope local --agent pi --project .
```

Preview the target paths first:

```bash
brain skills targets --scope both --agent codex --agent claude --agent copilot --agent pi --project .
```

`brain skills` installs only the Brain skill from this repo.

## Brainstorm To Planning To Execution

Brain is intentionally opinionated here:

1. brainstorm the high-level what and why
2. promote the brainstorm into an epic
3. build and approve the epic's canonical spec
4. break the approved spec into execution stories
5. complete stories while keeping the spec as the source of truth

In a project directory:

```bash
brain init --project .
brain doctor --project .
brain context install --project .
brain plan init --project .

brain brainstorm start --project . "Newsletter system"
brain plan epic promote --project . newsletter-system
brain plan spec show --project . newsletter-system
brain plan spec status --project . newsletter-system --set approved
brain plan story create --project . newsletter-system "Template editor"
brain plan story update --project . template-editor --status in_progress
brain plan status --project .
```

Use `brain adopt --project .` instead of `brain init --project .` when the repo already has docs or an unmanaged `AGENTS.md`.

## Quick Start

In any project directory:

```bash
brain init --project .
brain doctor --project .
brain context install --project .
brain plan init --project .
brain brainstorm start --project . "Initial ideas"
brain search --project . "architecture"
```

## Main Commands

- `brain init`: bootstrap a project-local Brain workspace
- `brain adopt`: adopt an existing repo into the Brain managed context model
- `brain doctor`: validate local Brain setup
- `brain read`, `brain edit`: inspect and update managed markdown
- `brain find`, `brain search`: project-local retrieval
- `brain brainstorm ...`: project-local brainstorming
- `brain plan ...`: project-local epic/spec/story planning
- `brain context ...`: install or refresh project context files
- `brain session ...`: enforce workflow and verification rules
- `brain skills ...`: install the Brain skill for agent runtimes
- `brain history`, `brain undo`: inspect and revert tracked note changes
- `brain version`, `brain update`: inspect or update the CLI

## Search Model

`brain` indexes project-managed markdown only:

- `AGENTS.md`
- `docs/**/*.md`
- `.brain/**/*.md`

It excludes local runtime state such as:

- `.brain/state/**`
- `.brain/sessions/**`

This keeps retrieval focused on durable project knowledge instead of transient internals.

By default, `brain` uses the built-in `localhash` provider. That gives you strong local lexical search plus lightweight semantic hinting without any network dependency. If you want stronger semantic retrieval, switch the embedding provider to `openai`.

## Config

Config lives at `~/.config/brain/config.yaml`.

Supported fields:

- `embedding_provider`
- `embedding_model`
- `output_mode`

Default values:

- `embedding_provider: localhash`
- `embedding_model: hash-v1`

Environment overrides:

- `BRAIN_EMBEDDING_PROVIDER`
- `BRAIN_EMBEDDING_MODEL`
- `BRAIN_OUTPUT_MODE`

Project state is derived from `--project` and `.brain/state`. It is not configured globally.

## Update Model

`brain update` downloads the newest matching GitHub Release, verifies checksums, and installs the binary.

By default, that means the latest stable release published automatically from `main`.

- if the current binary is writable, it updates in place
- otherwise it installs to:
  - Unix: `~/.local/bin/brain`
  - Windows: `%LocalAppData%\Programs\brain\brain.exe`
- replaced binaries are backed up under the global Brain app data directory

## Read Next

- [`docs/usage.md`](docs/usage.md)
- [`docs/architecture.md`](docs/architecture.md)
- [`docs/skills.md`](docs/skills.md)
- [`docs/why.md`](docs/why.md)
