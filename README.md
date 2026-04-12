# 🧠 brain

## Give Your AI Coding Agent A Real Brain Inside The Repo

`brain` is a local-first memory, context, planning, and retrieval layer for AI coding agents.

It gives every project its own durable brain inside the repo, so the agent stops starting from scratch, stops wasting turns rediscovering context, and works more reliably as the codebase evolves.

- Durable project memory in plain markdown
- Spec-driven workflow from brainstorming to shipped code
- Lower token spend and less tool sprawl by keeping everything local and integrated

## The Problem

AI coding agents are powerful, but they are stateless by default.

That usually means:

- repeated prompting just to restore project context
- stale assumptions about architecture and product decisions
- extra tokens spent rediscovering what the repo already knows
- planning that lives in separate tools instead of next to the code
- weak continuity across sessions, branches, and feature work

`brain` fixes that by making the project itself the memory system.

## Who This Is For

`brain` is for developers who already use AI coding agents heavily and are tired of paying the context tax every session.

It fits best when:

- you use an AI agent regularly on a real, evolving codebase
- you are tired of repeating the same architecture and product context
- you want planning, docs, retrieval, and workflow to live with the code
- you want the project to stay understandable to humans too, not just the agent

## What Brain Actually Does

`brain` keeps human docs at the repo root, machine-managed state under `.brain/`, and a local SQLite search index for durable project knowledge.

It provides explicit workflows for:

- docs and project context
- brainstorming
- epic -> spec -> story planning
- local retrieval
- note history and undo
- session enforcement for verification and durable updates

This is not another hosted dashboard, cloud vector database, or external planning layer. It lives with the project and is built specifically to work well with AI agents.

## Elevator Pitch

`brain` gives every software project its own local brain for AI agents: docs, context, planning, history, and search, all stored with the repo. It helps agents stop starting from scratch, stop wasting turns rediscovering context, and work more reliably in evolving codebases.

## Why It Helps

When an agent can read a real project contract, search durable project knowledge, follow a planning workflow, and update the right notes as the code evolves, you get a tighter and cheaper development loop.

Instead of stitching together n8n, cloud vector storage, external planning tools, and fragile chat history, `brain` keeps the important layer local:

- fewer moving parts
- fewer subscriptions and hosted dependencies
- less prompt repetition
- lower token waste
- better continuity between sessions

It does not make software work magic. It gives the agent a stable operating memory so it makes better decisions with less thrash.

## Mental Model

Every project gets its own Brain:

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

`brain skills` installs only the Brain skill from this repo. That gives the agent a ready-made way to use Brain correctly instead of treating the project brain like random files.

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

## Why Brain Exists

`brain` exists because most AI agent pain is not raw coding ability. It is continuity failure.

Agents can write code quickly, but they lose project context, drift on decisions, forget why work was scoped a certain way, and burn money re-learning the same repo over and over. Brain exists to keep that continuity local, durable, and usable by both the agent and the human team.

## Why This Saves Time And Money

`brain` cuts waste in two places:

- token waste from repeatedly feeding the same project context back into the agent
- tool waste from spreading memory, planning, retrieval, and workflow across too many separate systems

You still need good prompts and good engineering judgment. But when the agent can work against a stable local brain, you spend less money and fewer turns just rebuilding context.

## Explore Brain In More Detail

### How Brain Works

`brain` keeps markdown as the source of truth and uses local SQLite state for retrieval and operational support. Human-facing docs stay readable, while the agent gets a structured project contract, generated context, planning notes, and local search without any central service dependency.

Deep dive: [`docs/architecture.md`](docs/architecture.md)

### Using Brain Day To Day

The core operating loop is simple: install or adopt Brain in a repo, install the Brain skill, retrieve context with local search, move ideas into epics/specs/stories, and enforce verification plus durable note updates through sessions when you want a stricter workflow.

Deep dive: [`docs/usage.md`](docs/usage.md)

### Brain Skill

The Brain skill is the generic fallback that helps an agent immediately understand how to operate against a Brain-managed repo. It teaches the agent to read the project contract first, use Brain search and note workflows, and respect the repo’s planning and session model.

Deep dive: [`docs/skills.md`](docs/skills.md)

### Why This Model

The design is opinionated on purpose: one brain per project, plain markdown first, local search instead of centralized memory, and explicit contracts instead of hidden magic. That keeps the system portable, understandable, and resilient as a codebase evolves.

Deep dive: [`docs/why.md`](docs/why.md)

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
