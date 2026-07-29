# Architecture

`brain` is a single Go CLI with a project-local workspace model.

The architecture exists to support one product claim: every project gets its own durable local brain for AI agents. Markdown stays canonical, local SQLite powers retrieval, and the CLI exposes explicit workflows for context compilation, history, and execution discipline.

The repo has four important layers:

1. workspace and notes
2. indexing, retrieval, and safety
3. context compilation, session enforcement, and upgrade-aware repo guidance
4. optional compiled-module registration and project runtime state

## Workspace Model

The current project root is the primary boundary.

- `internal/workspace` owns path resolution, workspace validation, and markdown walking
- `internal/notes` owns frontmatter parsing, templates, note create/read/update/move behavior, and editor flow
- `internal/history` and `internal/backup` own append-only history, backups, and undo

The workspace only treats these locations as durable Brain-managed knowledge:

- `AGENTS.md`
- `docs/`
- `.brain/`

## Search And Indexing

- `internal/index` owns SQLite schema, chunking, FTS, and embedding persistence
- `internal/search` owns lexical plus semantic reranking
- `internal/embeddings` owns provider selection and embedding generation

The index is local to each project under `.brain/state/brain.sqlite3`.

## Product Systems

- `internal/projectcontext` generates and refreshes `AGENTS.md`, `.brain/context/*`, and `.brain/policy.yaml`, and it can integrate Brain-managed sections into existing local agent instruction files
- `internal/taskcontext` owns the summary-first context compiler and packet assembly for `brain context compile`
- `internal/structure` derives boundary, entrypoint, config-surface, and test-surface data for compiler consumers
- `internal/livecontext` inspects worktree, session, and verification-adjacent signals for live task context
- `internal/session` enforces preflight and closeout workflow rules and records packet telemetry
- `internal/distill` turns active session work into review-first durable-memory proposals
- `internal/promotion` classifies durable-memory candidates for closeout and distillation
- `internal/modules` owns compiled module descriptors, registration, project
  enablement/configuration, local permission grants, lifecycle, and health
- `internal/skills` installs the Brain skill into agent runtimes
- `internal/update` owns version/update behavior

## Composition Root

- `main.go` boots Cobra
- `cmd/*` stays thin and maps flags/args to internal services
- `internal/app` wires config, workspace, notes, search, context compilation,
  sessions, distillation, skills, the module runtime, and output

## Module Foundation

Phase 1 implements the minimal internal module foundation in `internal/modules`.
Compiled registrations enter through `app.Options`, while `brain modules`
provides list/show, grant/revoke, enable/disable, and health commands. Tracked
project intent lives in the project module configuration file; ignored local
permission grants live in Brain's state directory.

The production binary registers `official.planning` at the outer composition
root, disabled by default. The module proves controlled command/event
declarations, local `.plan/` adapter boundaries, permissions, lifecycle, health,
and disabled-state isolation without making Planning part of Core.

Later stages remain:

1. broader official Planning compatibility and adapters
2. future external-process community modules over a versioned protocol
3. future Brain Cloud module services and trusted web surfaces

Planning is the first official optional module. It must use formal Core contracts;
Core must remain useful without it. Existing packages stay in place initially.
The minimal migration uses one flat `internal/modules` package plus a test-only
module before any Planning domain extraction. A wholesale Core package-tree
reorganization is not part of the architecture.

Detailed contracts: `docs/modules/architecture.md` and
`docs/modules/planning-driven-requirements.md`.

## Key Design Rules

- project-local markdown is the source of truth
- search is derived state, not canonical state
- generated context must be deterministic and refreshable
- agent workflows should use explicit CLI operations instead of ad hoc file conventions
- session enforcement is the hard control layer above the softer context layer
- modules use registered Core facades rather than private storage or concrete app managers
- module enablement never implies permission grant or data deletion
- Planning memory updates are proposals unless an explicit Core policy grants more
