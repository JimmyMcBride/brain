# Why

## One Brain Per Project

Project knowledge ages badly when it lives in one global heap. `brain` now treats every repo as its own operating environment so the docs, planning, context, and search index belong to that project instead of to one universal memory pool.

## Plain Markdown First

Humans should be able to read and edit the durable state without a proprietary interface. `brain` keeps the canonical layer in markdown and uses SQLite, logs, and backups as derived local state.

## Search Without Centralization

Retrieval is still useful, but it does not need a shared global database. A per-project index keeps results focused, avoids accidental cross-project contamination, and makes the system easier to reason about.

## Explicit Agent Contracts

Agents behave better when the repo exposes a clear contract. `AGENTS.md`, `.brain/context/*`, and `.brain/policy.yaml` make the expected workflow visible and refreshable.

## Safe Workflow Over Magic

`brain` favors explicit commands, local state, history, undo, and recorded verification over implicit background behavior. The tool should be understandable under pressure.
