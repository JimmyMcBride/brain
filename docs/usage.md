# Usage

This is the practical operating guide for `brain` after install. Use it when you already understand the top-level pitch and want the day-to-day commands for adopting Brain in a repo, compiling task context, retrieving local memory, and running the session workflow.

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

This is the moment where a repo gets its local brain: contract, docs, generated context, and local state.

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

This is the core cost-saving loop. When the agent can retrieve local project knowledge instead of having it re-pasted into prompts, you spend fewer turns reconstructing context.

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

## Distillation

Use distillation when session work or a brainstorm should become proposed durable memory without mutating the destination notes directly.

```bash
brain distill --project . --session
brain distill --project . --brainstorm .brain/brainstorms/event-follow-up-ideas.md
brain brainstorm distill --project . .brain/brainstorms/event-follow-up-ideas.md
```

`brain distill --session` requires an active session and creates a proposal note under `.brain/resources/changes/` with source provenance, promotion-review diagnostics, and suggested markdown updates for review.

`brain distill --brainstorm ...` uses the same proposal flow for brainstorms. `brain brainstorm distill ...` remains supported as a compatibility wrapper.

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

Compile task context first, then reach for the compatibility views only when you need them:

```bash
brain context compile --project . --task "auth flow"
brain context explain --project . --last
brain context stats --project .
brain context install --project .
brain context refresh --project .
brain context refresh --project . --agent claude
brain context refresh --project . --dry-run
brain context structure --project .
brain context structure --project . --path internal/search
brain context structure status --project .
brain context live --project . --task "auth flow"
brain context live --project . --explain
brain context assemble --project . --task "auth flow"
brain context assemble --project . --explain
brain context load --project . --level 0
brain context load --project . --level 1
brain context load --project . --level 2
brain context load --project . --level 3 --query "auth flow"
```

Use `--force` when adopting an existing unmanaged `AGENTS.md` or docs file into the managed-block model.

`context compile` is the summary-first working-set compiler:

- resolves the task from `--task` or the active session
- emits the smallest justified packet Brain currently knows how to build: base contract, changed files, touched boundaries, nearby tests, top durable note summaries, verification hints, ambiguities, and provenance
- keeps boundary-aware context visible by carrying adjacency, responsibilities, and nearby-test relations into the packet
- ranks verification hints into strong or suggested command guidance with explicit source provenance
- applies conservative local utility adjustments only after repeated repo-local evidence such as later expansions, successful verification linkage, or durable-update linkage
- keeps included context in summary form with exact anchors and explicit inclusion reasons
- records packet metadata into the active session when a session is present, but still works normally without a session
- is the best first choice when you want one compact startup packet instead of a full static bundle or a broader explain-oriented assembly view

`context explain` and `context stats` are analysis surfaces for the compiler:

- `context explain --last` inspects the latest recorded packet, its included items, later expansions, and downstream outcomes such as verification runs, durable updates, and closeout status
- `context explain --packet <hash>` lets you inspect an older packet when you need to debug a specific compile result
- `context stats` summarizes likely signal items, likely noise items, repeated expansion patterns, and common verification links from local compiler telemetry
- both commands stay grounded in recorded packet metadata and local session telemetry rather than opaque remote analytics
- use them when tuning context quality or debugging ranking behavior, not as a replacement for normal `context compile` usage

`context structure` is the structural repo inspection surface:

- returns grouped boundaries, entrypoints, config surfaces, and test surfaces
- auto-rebuilds the derived structural cache when it is missing or stale
- supports `--path` to focus on one subtree
- `context structure status` reports freshness and counts without rebuilding

`context live` is the live-work inspection surface:

- resolves the task from `--task` or the active session
- returns an on-demand packet with task, session, changed-file, touched-boundary, nearby-test, verification, policy-hint, and ambiguity sections
- adds rationale and missing-signal reporting with `--explain`
- does not persist live state to SQLite or the session file
- derives repo-observable verification recipes from policy, Makefile targets, package scripts, CI workflows, and bounded recent successful session commands when they exist
- reports recent recorded session commands plus verification-profile satisfaction when a session is active
- only emits policy hints for strong-match conditions such as missing verification or missing durable note updates after repo changes

`context assemble` is the task-focused packet interface:

- resolves the task from `--task` or the active session
- assembles typed context from durable notes, generated context, structural repo context, live-work signals, and workflow/policy sources
- shows ambiguities and confidence for the current task packet
- adds rationale, omitted-nearby context, and missing-group reporting with `--explain`
- remains useful when you want the broader packet shape and explain surfaces during the compiler transition

`context load` is the legacy compatibility path:

- level 0 loads the AGENTS summary plus current state
- level 1 adds overview and workflows
- level 2 loads the full static context bundle
- level 3 adds search-injected relevant context, using `--query` or the active session task

`context install` and `context refresh` manage the root contract plus `.brain/context/*`. They do not create missing agent-specific instruction files.

If an existing agent file already contains a Brain-managed section, `context refresh --agent ...` updates that section in place. If the file is missing or has no Brain-managed section yet, `context refresh` leaves it alone.

If you want the full existing-repo bootstrap instead of just context takeover, use:

```bash
brain adopt --project .
brain adopt --project . --agent codex
```

`adopt` scans for existing local agent instruction files such as `.codex/AGENTS.md`, `.claude/CLAUDE.md`, or `.pi/AGENTS.md` and appends or updates a Brain-managed section inside them while preserving the rest of the file. `adopt --agent ...` is the explicit path that may create a missing agent instruction file, and unsupported agent names are rejected instead of creating ad hoc paths.

## Sessions

Use sessions when the repo should require explicit verification and durable note updates.

```bash
brain session start --project . --task "tighten auth flow"
brain session validate --project .
brain session run --project . -- go test ./...
brain session run --project . -- go build ./...
brain session finish --project . --summary "auth flow tightened"
```

If finish blocks because repo changes need durable memory updates, run `brain distill --project . --session`, review the proposal, apply the note updates that matter, and retry `brain session finish`.
The blocked closeout output also surfaces packet-backed promotion suggestions when Brain can justify them. Review those suggestions first, then use the distill note to decide what should actually become durable memory.

## Skills

Install the Brain skill when you want the agent runtime itself to understand how to use the repo brain correctly from the start.

```bash
brain skills targets --scope both --agent codex --project .
brain skills install --scope global --agent codex
brain skills install --scope local --agent codex --project .
brain skills install --scope global --agent claude
brain skills install --scope global --agent copilot
brain skills install --scope local --agent copilot --project .
brain skills install --scope global --agent pi
brain skills install --scope local --agent pi --project .
brain skills install --scope global --agent openclaw
brain skills install --scope local --agent openclaw --project .
```

`brain skills install` always installs the Brain skill. Use `--scope global` to add it to your machine and `--scope local --project .` to add it to the current project.

The Brain skill is bundled into the running binary, so `brain skills install` and `brain skills targets` work from any directory instead of depending on a nearby Brain source checkout.

Brain always copies skill directories. It does not symlink them.

Installed skills include a generated `.brain-skill-manifest.json` file beside `SKILL.md`. Brain uses that manifest to detect stale or legacy local installs and repair them automatically the next time you work in that project.

When a branch changes Brain's command surface or agent-facing workflow guidance, update `skills/brain/SKILL.md` in that same branch and reinstall the local Brain skill for Codex and OpenClaw before closing the work:

```bash
go run . skills install --scope local --agent codex --agent openclaw --project .
```

Then reinstall or refresh with the installed binary:

```bash
brain skills install --scope local --agent codex --agent openclaw --project .
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

On Windows, `brain update` uses the same release assets and default install target as `scripts/install.ps1`.

By default, `brain update` tracks the latest stable GitHub release published from `main`.

When you run `brain update`, Brain refreshes any already-installed global Brain skills plus any local Brain skills inside the current `--project`. Other project-local installs repair themselves lazily the next time Brain runs in those repos.
