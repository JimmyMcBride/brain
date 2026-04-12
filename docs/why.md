# Why

`brain` exists because AI coding agents are strong at local execution but weak at continuity.

Without durable project memory, they keep rediscovering context, repeating planning work, and drifting as the codebase evolves. `brain` makes the repo itself the memory system so docs, planning, context, and search all stay with the project instead of being scattered across chat history and external tools.

## One Brain Per Project

`brain` treats every repo as its own operating environment so the docs, planning, context, and search index belong directly to that project.

## Plain Markdown First

Humans should be able to read and edit the durable state without a proprietary interface. `brain` keeps the canonical layer in markdown and uses SQLite, logs, and backups as derived local state.

## Search Without Centralization

Retrieval should stay focused on the project you are in. A per-project index keeps results relevant, avoids cross-project contamination, and makes the system easier to reason about.

It also removes the need for a separate hosted memory layer when local project search is enough.

## Explicit Agent Contracts

Agents behave better when the repo exposes a clear contract. `AGENTS.md`, `.brain/context/*`, and `.brain/policy.yaml` make the expected workflow visible and refreshable.

## Safe Workflow Over Magic

`brain` favors explicit commands, local state, history, undo, and recorded verification over implicit background behavior. The tool should be understandable under pressure.
