# Why

## One Brain Per Project

`brain` treats every repo as its own operating environment so the docs, planning, context, and search index belong directly to that project.

## Plain Markdown First

Humans should be able to read and edit the durable state without a proprietary interface. `brain` keeps the canonical layer in markdown and uses SQLite, logs, and backups as derived local state.

## Search Without Centralization

Retrieval should stay focused on the project you are in. A per-project index keeps results relevant, avoids cross-project contamination, and makes the system easier to reason about.

## Explicit Agent Contracts

Agents behave better when the repo exposes a clear contract. `AGENTS.md`, `.brain/context/*`, and `.brain/policy.yaml` make the expected workflow visible and refreshable.

## Safe Workflow Over Magic

`brain` favors explicit commands, local state, history, undo, and recorded verification over implicit background behavior. The tool should be understandable under pressure.
