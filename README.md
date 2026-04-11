# brain

`brain` is a local-first CLI that gives each software project its own markdown-based operating layer.

It keeps human docs at the repo root, keeps machine-managed state under `.brain/`, builds a local SQLite search index, and provides explicit workflows for planning, brainstorming, context, history, and session enforcement.

## What It Does

- initializes a project-local Brain workspace in any repo or folder
- keeps durable project knowledge in plain markdown
- indexes `AGENTS.md`, `docs/**/*.md`, and `.brain/**/*.md` with SQLite FTS plus embeddings
- provides project-scoped planning and brainstorming commands
- generates deterministic agent context and optional wrappers
- tracks note history and supports undo
- enforces repo workflows through session policy

## Mental Model

`brain` is not a shared global memory store anymore. Every project gets its own Brain.

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
- `.brain/planning/` holds epics/stories or other planning structures.
- `.brain/brainstorms/` holds project-local ideation notes.
- `.brain/resources/` holds durable references, captures, and change history.
- `.brain/state/` holds SQLite, history logs, backups, and other local state.

## Install

### Build from source

```bash
git clone https://github.com/JimmyMcBride/brain.git
cd brain
go build -o brain .
install -Dm0755 brain ~/.local/bin/brain
```

### Go run during development

```bash
go run . --help
```

Use `go run .` when working on the CLI itself before replacing the installed binary.

## Quick Start

In any project directory:

```bash
brain init --project .
brain doctor --project .
brain context install --project .
brain plan init --project . --paradigm epics
brain brainstorm start --project . "Initial ideas"
brain search --project . "architecture"
```

## Main Commands

- `brain init`: bootstrap a project-local Brain workspace
- `brain doctor`: validate local Brain setup
- `brain read`, `brain edit`: inspect and update managed markdown
- `brain find`, `brain search`: project-local retrieval
- `brain brainstorm ...`: project-local brainstorming
- `brain plan ...`: project-local planning and work tracking
- `brain context ...`: install or refresh project context files
- `brain session ...`: enforce workflow and verification rules
- `brain skills ...`: install the Brain skill bundle for agent runtimes
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

## Config

Config lives at `~/.config/brain/config.yaml`.

Supported fields:

- `embedding_provider`
- `embedding_model`
- `output_mode`

Environment overrides:

- `BRAIN_EMBEDDING_PROVIDER`
- `BRAIN_EMBEDDING_MODEL`
- `BRAIN_OUTPUT_MODE`

Project state is derived from `--project` and `.brain/state`. It is not configured globally.

## Update Model

`brain update` downloads the newest matching GitHub Release, verifies checksums, and installs the binary.

- if the current binary is writable, it updates in place
- otherwise it installs to `~/.local/bin/brain`
- replaced binaries are backed up under the global Brain app data directory

## Read Next

- [`docs/usage.md`](docs/usage.md)
- [`docs/architecture.md`](docs/architecture.md)
- [`docs/skills.md`](docs/skills.md)
- [`docs/why.md`](docs/why.md)
