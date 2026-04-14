---
created: "2026-04-13T22:00:56Z"
epic: structural-repo-context
project: brain
status: approved
title: Structural Repo Context Spec
type: spec
updated: "2026-04-13T22:34:08Z"
---
# Structural Repo Context Spec

Created: 2026-04-13T22:00:56Z

## Why

Brain currently has durable markdown context and deterministic generated docs, but it lacks a cheap structural layer that helps an agent orient around the repository before deeper retrieval. A lightweight repo map is the most practical next step.

## Problem

Without structural repo context, Brain has to infer too much from docs and note retrieval. That makes the system weaker at quickly identifying likely boundaries, entrypoints, test surfaces, and adjacent areas of code that matter for a task.

## Goals

- Add a compact derived structural layer that improves orientation and context selection.
- Keep the first wave language-agnostic and deterministic.
- Prefer useful repo boundaries and entrypoints over pretending Brain has deep parser-grade code understanding.
- Feed reusable structural signals into task-context assembly.
- Make the structural layer directly inspectable instead of burying it inside ranking heuristics.

## Non-Goals

- Building a full semantic code graph.
- Shipping deep language-specific symbol analysis in the first wave.
- Replacing direct code reading with structural summaries.
- Expanding the canonical truth model beyond markdown.
- Implementing incremental structural refresh in this epic.

## Requirements

- Add `brain context structure` as the direct inspection command for structural repo context.
- Add `brain context structure status` as the structural freshness and debug surface.
- Store structural repo context as derived data in `.brain/state/brain.sqlite3`.
- Give structural repo context its own freshness lifecycle, separate from markdown-search freshness.
- Derive a language-agnostic repo map focused on boundaries, entrypoints, config surfaces, and test surfaces.
- Keep the first wave limited to boundaries and entry surfaces; do not add function, class, or symbol extraction yet.
- Make structural outputs reusable by task-context assembly through the `structural_repo` source group defined in the Task Context Assembly spec.

## UX / Flows

Direct inspection flow:
1. User runs `brain context structure`.
2. Brain checks the structural cache freshness.
3. If the cache is missing or stale, Brain rebuilds it fully.
4. Brain returns a compact grouped repo map with boundaries, entrypoints, config surfaces, and test surfaces.

Filtered inspection flow:
1. User runs `brain context structure --path internal/search`.
2. Brain loads the structural cache.
3. Brain returns only structural items under that subtree.

Status flow:
1. User runs `brain context structure status`.
2. Brain returns structural freshness metadata and counts without mutating the cache.

Task-context integration flow:
1. User runs `brain context assemble ...`.
2. Brain reads the structural cache.
3. Brain uses structural items as candidates for the `structural_repo` packet group.

## Data / Interfaces

Public command surface:
- `brain context structure`
- `brain context structure --path <prefix>`
- `brain context structure status`

Human output sections for `context structure`:
- `## Repository Shape`
- `## Boundaries`
- `## Entrypoints`
- `## Config Surfaces`
- `## Test Surfaces`

JSON output contract for `context structure`:

```json
{
  "summary": {
    "runtime": "go",
    "item_count": 0,
    "boundary_count": 0,
    "entrypoint_count": 0,
    "config_surface_count": 0,
    "test_surface_count": 0
  },
  "boundaries": [],
  "entrypoints": [],
  "config_surfaces": [],
  "test_surfaces": []
}
```

Structural item shape:

```json
{
  "kind": "boundary",
  "path": "internal/search/",
  "label": "internal/search",
  "role": "library",
  "summary": "Go package area under internal/",
  "evidence": ["matched common source root", "contains Go files"]
}
```

Required derived tables in `.brain/state/brain.sqlite3`:
- `structure_state`
  - `indexed_at`
  - `workspace_signature`
  - `indexed_file_count`
  - `item_count`
  - `boundary_count`
  - `entrypoint_count`
  - `config_surface_count`
  - `test_surface_count`
- `structure_items`
  - `id`
  - `kind`
  - `path`
  - `label`
  - `role`
  - `summary`
  - `evidence_json`

Structural item kinds:
- `boundary`
- `entrypoint`
- `config_surface`
- `test_surface`

Role values:
- `app`
- `library`
- `config`
- `tests`
- `scripts`
- `docs`
- `brain`
- `ci`
- `unknown`

## Freshness / Lifecycle

Structural freshness states:
- `missing`
- `stale`
- `fresh`

Structural freshness reasons:
- `structure metadata missing`
- `workspace signature changed`
- `workspace matches`

Behavior:
- `brain context structure status` only reports state and metadata.
- `brain context structure` auto-syncs by rebuilding fully when state is `missing` or `stale`.
- Structural freshness is independent from markdown-search freshness.
- This epic uses full rebuild on stale state and does not introduce incremental updates.

## Scanner Contract

Introduce a dedicated structural subsystem rather than hiding the model inside generated-doc code.

Scanner exclusions:
- `.git/`
- `.brain/state/`
- `.brain/sessions/`
- `node_modules/`
- `vendor/`
- `.venv/`
- `venv/`
- `dist/`
- `build/`
- `coverage/`
- `.next/`

First-wave detection rules:

Boundary items:
- important root directories except ignored/generated ones
- first-level subdirectories under:
  - `cmd/`
  - `internal/`
  - `pkg/`
  - `src/`
  - `app/`
  - `services/`
  - `lib/`
  - `scripts/`
  - `config/`
  - `test/`
  - `tests/`

Entrypoint items:
- root `main.*`
- `cmd/*/main.*`
- runtime manifests:
  - `go.mod`
  - `package.json`
  - `Cargo.toml`
  - `pyproject.toml`
- top-level bootstrap files named `main.*` under `app/` or `src/`

Config-surface items:
- `.github/workflows/*`
- `config/` subtree summary
- root config/manifests:
  - `go.mod`
  - `package.json`
  - `Cargo.toml`
  - `pyproject.toml`
  - `Makefile`
  - `.env*`
  - root `*.yaml`, `*.yml`, `*.json`, `*.toml`

Test-surface items:
- `test/`, `tests/`, `spec/` directories
- files matching:
  - `*_test.go`
  - `*.test.*`
  - `*.spec.*`
- summarize test surfaces by directory or boundary rather than emitting every test file as its own top-level structural item unless that is the only detectable test surface

Role inference rules:
- `app`: `cmd/`, `app/`, root `main.*`, executable entry roots
- `library`: `internal/*`, `pkg/*`, `lib/*`, `src/*`
- `config`: `config/`, manifests, env/config files
- `tests`: test dirs and test surfaces
- `scripts`: `scripts/`
- `docs`: `docs/`
- `brain`: `.brain/`
- `ci`: `.github/workflows/`
- `unknown`: any unmatched path

## Task Context Integration

This epic populates the reserved `structural_repo` group in Task Context Assembly.

Contract with task assembly:
- structural items must be consumable by `path`, `label`, `role`, and `summary`
- task assembly may match over `path`, `label`, `role`, `summary`, and `evidence`
- this epic does not define packet ranking weights
- this epic defines structural context as compact orientation signals, not as file dumps or code summaries

## Risks / Open Questions

- Are the default path heuristics broad enough to work across mixed repos without creating noisy boundaries?
- Is `--path` filtering sufficient for first-wave debugging, or will users later need richer structural queries?
- How often will repos have weak enough structure that the first-wave map feels sparse?

## Rollout

1. Add the structural storage model and status lifecycle.
2. Add `brain context structure` and `brain context structure status`.
3. Implement first-wave detection for boundaries, entrypoints, config surfaces, and test surfaces.
4. Wire structural items into the `structural_repo` group for later task-assembly work.

## Story Breakdown

- [ ] Add the structural cache schema and freshness lifecycle.
- [ ] Add `brain context structure` and `brain context structure status`.
- [ ] Implement first-wave structural detection for boundaries, entrypoints, config surfaces, and test surfaces.

## Resources

- [[.brain/brainstorms/context-engine-v2.md]]
- [[.brain/resources/references/architecture-and-code-map.md]]
- [[.brain/resources/references/retrieval-and-indexing.md]]
- [[.brain/planning/specs/task-context-assembly.md]]

## Notes

The value of this epic is reusable compression and orientation, not code intelligence. If a structural detail cannot be derived reliably with the language-agnostic rules above, it should stay out of the first wave.
