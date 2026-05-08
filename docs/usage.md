---
updated: "2026-04-16T22:00:00Z"
---
# Usage

This is the practical operating guide for `brain` after install. Use it when you want the day-to-day commands for adopting Brain in a repo, compiling task context, retrieving local memory, and running the session workflow.

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

If no release has been published yet, the installer falls back to downloading the current `main` source tarball and building it locally with Go.

## Bootstrap A Project

This is the moment where a repo gets its local brain: contract, docs, generated context, and local state.

For a new or mostly empty repo:

```bash
brain init --project .
brain doctor --project .
brain context install --project .
```

For an existing repo that already has docs or an unmanaged `AGENTS.md`:

```bash
brain adopt --project .
brain doctor --project .
```

This creates:

- `AGENTS.md`
- `docs/project-overview.md`
- `docs/project-architecture.md`
- `docs/project-workflows.md`
- `.brain/context/*`
- `.brain/policy.yaml`
- `.brain/state/brain.sqlite3`

Brain treats `.brain/session.json`, `.brain/sessions/`, `.brain/state/`, and `.brain/policy.override.yaml` as local runtime state. They should stay out of Git by default while the durable shared layer lives in markdown and docs.

`brain init` is the clean bootstrap path.  
`brain adopt` is the existing-repo path: it creates the local Brain workspace, adopts Brain-owned docs into the managed-block model, and preserves previous content under `Local Notes`.

The generated files are starter context for AI agents, not complete repo memory. After `brain adopt`, the AI agent should scan repo structure, docs, manifests, entrypoints, tests, CI, config, and deployment surfaces, then update AGENTS.md, docs, or `.brain` notes with durable project-specific findings. Use focused `.brain/resources` notes for architecture, workflows, risks, and references that do not belong in top-level templates, and keep generated managed blocks refreshable by putting hand-authored findings in Local Notes or dedicated notes.

## Read And Update Notes

```bash
brain read --project . AGENTS.md
brain read --project . docs/project-overview.md
brain edit --project . docs/project-overview.md -b "# Project Overview\n\nUpdated body."
brain edit --project . AGENTS.md --editor nvim
```

Use Brain-managed markdown for durable context, decisions, references, and change notes. If you need scratch or product-management notes, keep those in the tools that already own them.

## Retrieve Context

This is the core cost-saving loop. When the agent can retrieve local project knowledge instead of having it re-pasted into prompts, you spend fewer turns reconstructing context.

```bash
brain find --project . auth
brain search --project . "Supabase auth"
brain search --project . status
brain search --project . --explain "Supabase auth"
brain search --project . --inject "Supabase auth"
```

`find` is path/title/type/content matching.  
`search` uses the local SQLite index plus the configured embedding provider over project-managed markdown. With the default `localhash` provider, the result is strong local lexical search plus lightweight semantic hinting rather than a high-quality hosted semantic model.

The index lives in `.brain/state/brain.sqlite3`, tracks its own freshness, rebuilds automatically when it is missing or stale, and `brain doctor` plus `brain search status` both show the active provider/model.

## Distillation

Use distillation when active session work should become proposed durable memory without mutating the destination notes directly.

```bash
brain distill --project . --session
brain distill --project . --session --dry-run
```

`brain distill --session --dry-run` requires an active session and prints the full proposal review without creating a repo-owned file.

`brain distill --session` requires an active session and creates a proposal note under `.brain/resources/changes/` with source provenance, promotion-review diagnostics, and suggested markdown updates for review.

Treat that proposal note as repo-owned worktree state: if it belongs to the active task, keep it in the same branch and PR; otherwise review it and intentionally remove it before returning to `develop`, `release/*`, or `main`.

## Context Management

Start task work with `brain prep`, then reach for the lower-level compiler or compatibility views only when you need them:

```bash
brain prep --project . --task "auth flow"
brain prep --project . --task "auth flow" --budget small
brain prep --project . --task "auth flow" --fresh
brain context compile --project . --task "auth flow"
brain context compile --project . --task "auth flow" --budget small
brain context compile --project . --task "auth flow" --budget 1200
brain context compile --project . --task "auth flow" --fresh
brain context explain --project . --last
brain context stats --project .
brain context effectiveness --project .
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

`brain prep` is the default startup path:

- starts a validated session when none exists and requires `--task` in that case
- reuses the active session when one already exists and validates it before compiling
- accepts the same `--budget` and `--fresh` controls as `context compile`
- errors instead of silently switching tasks when the requested `--task` does not match the active session
- prints the compiled packet plus the short next-step guidance Brain expects agents to follow

`context compile` remains the lower-level manual path when you need the packet compiler directly without the startup orchestration.

`context compile` is the summary-first working-set compiler:

- resolves the task from `--task` or the active session
- accepts `--budget small|default|large|<integer>` so you can ask for a tighter or wider startup packet without guessing what Brain will omit
- emits the smallest justified packet Brain currently knows how to build: base contract, changed files, touched boundaries, nearby tests, top durable note summaries, verification hints, ambiguities, and provenance
- uses deterministic local token-cost heuristics plus explicit reserve buckets for base contract, verification, and diagnostics before choosing optional working-set items
- automatically reuses the latest matching packet inside the active session and returns a compact reused response instead of reprinting unchanged packet sections wholesale
- emits a compact `delta` response with changed sections, changed item ids, and invalidation reasons when the task is stable but relevant compile inputs changed
- supports `--fresh` when you need to bypass reuse and force a standalone full packet for debugging or inspection
- reports target, used, remaining, reserve, omitted-candidate budget diagnostics, and reuse or delta lineage in compile output and `context explain`
- records full packet bodies plus lineage metadata into the active session when a session is present, but still works normally without a session

`context explain`, `context stats`, and `context effectiveness` are analysis surfaces for the compiler:

- `context explain --last` inspects the latest recorded packet, including cache status, reuse or delta lineage, invalidation reasons, included items, later expansions, post-packet searches, Brain-routed context access, and downstream outcomes such as verification runs, durable updates, and closeout status
- `context explain --packet <hash>` lets you inspect an older packet when you need to debug a specific compile result
- `context stats` summarizes likely signal items, likely noise items, repeated expansion patterns, common verification links, fresh-packet budget-pressure frequency, and recurring omitted markdown docs from local compiler telemetry
- `context effectiveness` turns packet telemetry into a higher-level report on packet usage, cache behavior, budget pressure, post-packet searches, Brain-routed context reads/searches, likely misses from omitted docs later accessed, telemetry gaps, and recommended packet-shaping follow-ups

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

`context assemble` is the broader typed packet interface when you want more than the compiler-first startup packet.

`context load` is the older compatibility path for static-bundle style context loading.

`context install` and `context refresh` manage the root contract plus `.brain/context/*`. They do not create missing agent-specific instruction files.

If you want the full existing-repo bootstrap instead of just context takeover, use:

```bash
brain adopt --project .
brain adopt --project . --agent codex
```

There is no separate `brain --adopt` flag; use the `brain adopt` command.

## Sessions

Use sessions when the repo should require explicit verification and durable note updates.

```bash
brain prep --project . --task "tighten auth flow"
brain session run --project . -- go test ./...
brain session run --project . -- go build ./...
brain session finish --project . --summary "auth flow tightened"
```

`brain session validate`, `brain session start`, and `brain context compile` all still work directly, but `brain prep` is the normal first reflex because it validates or starts the session and compiles the startup packet in one step.

If finish blocks because repo changes need durable memory updates, inspect the promotion suggestions first. Run `brain distill --project . --session --dry-run` when you need the full review without creating a proposal note; use `brain distill --project . --session` only when you intentionally want a tracked proposal note, then apply the note updates that matter and retry `brain session finish`.

## History And Undo

```bash
brain history --project .
brain undo --project .
```

These operate on Brain-managed note changes recorded in the local history log.

## Project Upgrades

Use `brain doctor --project .` to inspect whether project migrations are `current`, `pending`, or `broken`. `doctor` may also report that explicit runtime-state cleanup is available without treating the repo as broken.

Use `brain update --project .` to refresh the current Brain binary, refresh already-installed Brain skills, and apply pending project migrations for the current repo when appropriate.

Use `brain context migrate --project .` when you want to run the project migration path explicitly with the current binary.

Lazy preflight repair only applies auto-safe migrations. Git-index cleanup for ignored Brain runtime state is explicit-only, so it runs through `brain update --project .` or `brain context migrate --project .`, prints what it changed, and leaves the resulting diff for you to review and commit.
