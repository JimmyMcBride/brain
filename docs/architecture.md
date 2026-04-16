# Architecture

`brain` is a single Go CLI with a project-local workspace model.

The architecture exists to support one product claim: every project gets its own durable local brain for AI agents. Markdown stays canonical, local SQLite powers retrieval, and the CLI exposes explicit workflows for context compilation, history, and execution discipline.

The repo has three important layers:

1. workspace and notes
2. indexing, retrieval, and safety
3. context compilation, session enforcement, and upgrade-aware repo guidance

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
- `internal/skills` installs the Brain skill into agent runtimes
- `internal/update` owns version/update behavior

## Composition Root

- `main.go` boots Cobra
- `cmd/*` stays thin and maps flags/args to internal services
- `internal/app` wires config, workspace, notes, search, context compilation, sessions, distillation, skills, and output

## Key Design Rules

- project-local markdown is the source of truth
- search is derived state, not canonical state
- generated context must be deterministic and refreshable
- agent workflows should use explicit CLI operations instead of ad hoc file conventions
- session enforcement is the hard control layer above the softer context layer
