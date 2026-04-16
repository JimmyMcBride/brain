# 🧠 brain

![Scarecrow dancing](docs/assets/scarecrow.gif)

## Give Your AI Coding Agent A Real Brain Inside The Repo

`brain` is a local-first memory, context, retrieval, and workflow layer for AI coding agents.

It gives every project a durable operating memory inside the repo so the agent stops starting from scratch, stops wasting turns rediscovering context, and works more reliably as the codebase evolves.

- Durable project memory in plain markdown
- Compiled startup context with packet budgets and session reuse
- Local retrieval backed by project-local SQLite
- Session enforcement for verification and durable updates
- Note history and undo for Brain-managed markdown

## The Problem

AI coding agents are powerful, but they are stateless by default.

That usually means:

- repeated prompting just to restore project context
- stale assumptions about architecture and product decisions
- extra tokens spent rediscovering what the repo already knows
- weak continuity across sessions, branches, and feature work
- too much context living in chat history instead of the repo

`brain` fixes that by making the project itself the memory system.

## Who This Is For

`brain` fits best when:

- you use an AI coding agent regularly on a real, evolving codebase
- you want repo-local context instead of depending on chat history
- you want retrieval, compiled context, and workflow discipline to live with the code
- you want Brain to stay focused on memory and execution context instead of trying to own every part of software delivery

## What Brain Actually Does

`brain` keeps human docs at the repo root, machine-managed context under `.brain/`, and a local SQLite index for durable project knowledge.

It provides explicit workflows for:

- project contracts and docs
- local retrieval
- compiled task context
- session enforcement
- note history and undo
- promotion-style distillation from active work sessions

This is not another hosted dashboard, cloud vector database, or issue tracker. It lives with the project and is built specifically to help coding agents stay grounded in local truth.

## Mental Model

Every project gets its own Brain:

```text
my-project/
  AGENTS.md
  docs/
  .brain/
    context/
    resources/
    sessions/
    state/
```

- `AGENTS.md` is the root contract for humans and agents.
- `docs/` is the human-readable project documentation layer.
- `.brain/context/` is the generated modular context bundle.
- `.brain/resources/` holds durable references, captures, and change history.
- `.brain/sessions/` holds recorded session ledgers.
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

Stable GitHub releases are published from `main`. Prefer PR merges as the normal path into `main`, then install from the latest release published there.

If no GitHub release exists yet, the installer falls back to downloading the current `main` source archive from GitHub and building it locally with Go.

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

## Quick Start

In any project directory:

```bash
brain init --project .
brain doctor --project .
brain context install --project .
brain search --project . "architecture"
brain context compile --project . --task "auth flow"
brain session start --project . --task "tighten auth flow"
brain session run --project . -- go test ./...
brain session finish --project . --summary "auth flow tightened"
```

Use `brain adopt --project .` instead of `brain init --project .` when the repo already has docs or an unmanaged `AGENTS.md`.

## What Brain Does Not Try To Be

`brain` intentionally stays focused on repo-local memory and execution context.

It does not try to replace:

- your roadmap or issue tracker
- your issue tracker
- hosted product-management software
- cloud memory systems glued on top of the repo

If you already use separate delivery tools, Brain is designed to complement them rather than compete with them.

## Main Commands

- `brain init`: bootstrap a project-local Brain workspace
- `brain adopt`: adopt an existing repo into the Brain-managed context model
- `brain doctor`: validate local Brain setup
- `brain read`, `brain edit`: inspect and update managed markdown
- `brain find`, `brain search`: project-local retrieval
- `brain context ...`: install, refresh, compile, inspect, and analyze task context
- `brain distill --session`: create a reviewed distillation proposal from active session work
- `brain session ...`: enforce workflow and verification rules
- `brain skills ...`: install the Brain skill for agent runtimes
- `brain history`, `brain undo`: inspect and revert tracked note changes
- `brain version`, `brain update`: inspect or update the CLI

## Deep Dives

- [Usage](docs/usage.md)
- [Architecture](docs/architecture.md)
- [Skills](docs/skills.md)
- [Why](docs/why.md)
