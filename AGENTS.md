---
updated: "2026-04-20T05:35:38Z"
---
# Project Agent Contract

<!-- brain:begin agents-contract -->
Use this file as a Brain-managed project context entrypoint for `brain`.

Brain is intended for AI agents operating in this repo, not as a human-operated project dashboard.

Read the linked context files before substantial work. Prefer the `brain` skill and `brain` CLI for project memory, retrieval, and durable context updates.

## Table Of Contents

- [Overview](./.brain/context/overview.md)
- [Architecture](./.brain/context/architecture.md)
- [Standards](./.brain/context/standards.md)
- [Workflows](./.brain/context/workflows.md)
- [Memory Policy](./.brain/context/memory-policy.md)
- [Current State](./.brain/context/current-state.md)
- [Policy](./.brain/policy.yaml)

## Project Docs

- [README.md](./README.md)
- [architecture.md](./docs/architecture.md)
- [project-architecture.md](./docs/project-architecture.md)
- [project-overview.md](./docs/project-overview.md)
- [project-workflows.md](./docs/project-workflows.md)
- [skills.md](./docs/skills.md)
- [usage.md](./docs/usage.md)
- [why.md](./docs/why.md)

## Required Workflow

1. If no validated session is active, run `brain prep --task "<task>"`.
2. If a session is already active, run `brain prep`.
3. Read this file and the linked context files still needed for the task.
4. Use `brain context compile --task "<task>"` only when you need the lower-level packet compiler directly.
5. Retrieve project memory with `brain find brain` or `brain search "brain <task>"` when the compiled packet is not enough.
6. Use `brain edit` for durable context updates to AGENTS.md, docs, or .brain notes.
7. Use `brain session run -- <command>` for required verification commands.
8. Finish with `brain session finish` so policy checks can enforce verification and surface promotion review when durable follow-through is still needed.

## Post-Adoption Enrichment

After `brain adopt` creates starter context, the AI agent must scan the repo before treating the templates as complete memory.

1. Inspect repo structure, docs, manifests, entrypoints, tests, CI, config, and deployment surfaces.
2. Replace generic template notes with concrete project facts in AGENTS.md, docs, or .brain notes.
3. Add focused .brain/resources notes for architecture, workflows, risks, and references that do not belong in top-level templates.
4. Keep generated managed blocks refreshable; put hand-authored findings in Local Notes or dedicated notes.
<!-- brain:end agents-contract -->

## Local Notes

- 2026-04-20: Gitflow source of truth for this project is `develop` -> `release/vX.Y.Z` -> `main`.
- 2026-04-20: `develop`, `release/*`, and `main` are protected branches. Never push directly to them, never delete them, and land all changes through pull requests.
- 2026-04-20: Normal feature and bug-fix work should usually start from the latest `develop` line and open PRs back into `develop`.
- 2026-04-20: Official releases cut `release/vX.Y.Z` from `develop`, stabilize there, then merge `release/vX.Y.Z` into `main` to publish.
- 2026-04-20: Release-branch fixes must land in `develop` first and then be cherry-picked into the active `release/vX.Y.Z` branch.
- 2026-04-20: Production hotfixes may branch from the active `release/vX.Y.Z` branch or from `main`, whichever best matches production, but the equivalent fix must always end up in `develop`.
- 2026-04-20: After every PR merge into `develop`, fetch latest remote state, check out the updated `origin/develop`, and refresh Brain project context so repo memory tracks latest `develop`. If a repo-local `.plan/` workspace exists later, refresh that context from latest `develop` too.
- 2026-04-20: Before switching away from a working branch or returning to `develop`, `release/*`, or `main`, run `git status --short` and resolve repo-owned leftovers. If `.brain/resources/changes/*`, `.brain/`, `docs/`, or contract files belong to the task, merge them in the same branch/PR; otherwise review and intentionally remove them instead of carrying them onto a protected branch.
