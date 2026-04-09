# Why

## PARA At The Top

The top level stays strict because retrieval, automation, and human navigation all benefit from predictable roots. `brain` keeps the opinionated part small and lets richer conventions live below each PARA folder.

## Hybrid Retrieval

Keyword search alone misses related phrasing. Embeddings alone can drift or overgeneralize. Combining SQLite FTS5 with vector similarity gives a practical local retrieval stack that is inspectable, debuggable, and fast enough for personal knowledge workflows.

## Agent-First Design

AI agents work better when the system exposes explicit commands with stable side effects. `brain` provides structured creation, search, content packaging, history, and undo so agents can work with a vault safely instead of manipulating raw files blindly.

## Local-First Trust

The vault is plain markdown. Indexes, backups, and logs are local. The default embedder works offline. OpenAI embeddings are optional rather than required.

