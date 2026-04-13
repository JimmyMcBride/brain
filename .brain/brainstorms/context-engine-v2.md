---
brainstorm_status: active
created: "2026-04-13T22:00:29Z"
idea_count: "9"
project: brain
title: Context Engine V2 Direction
type: brainstorm
updated: "2026-04-13T22:02:55Z"
---
# Brainstorm: Context Engine V2 Direction

Started: 2026-04-13T22:00:29Z

## Focus Question

How should Brain improve its context-engineering capabilities so it assembles better task-specific context without weakening the markdown-first, local-first trust model?

## Ideas

- Keep `AGENTS.md`, `docs/`, and `.brain/` as the only durable canonical truth. Any repo maps, git/session signals, and future graphs stay derived and disposable.
- Optimize first for task-specific context quality, not bigger memory or more sources. The key product question is still: what context does this task need right now?
- Add typed context assembly so Brain can intentionally combine markdown truth, generated docs, structural repo context, live work signals, and repo policy instead of treating everything like note retrieval.
- Add a lightweight structural repo layer with repo tree summaries, entrypoints, config/test surfaces, module boundaries, and modest symbol hints where they are reliable.
- Add live work awareness from active sessions, current diff, touched files, nearby tests, recent verification results, and applicable policy or workflow guidance.
- Keep the first wave language-agnostic and deterministic. Do not overpromise parser-grade code intelligence yet.
- Prefer extending `brain context load` and related flows before adding a new flagship command. A dedicated packet command should only appear if the assembled output proves meaningfully different.
- Brain should explain why context was selected, what was omitted, and where ambiguity remains. Transparency is the advantage over more magical systems.
- Defer broad relationship-graph work, language-specific deep symbol analysis, and incremental indexing as separate follow-on initiatives until packet quality proves the need.

## Related

- [[.brain/planning/epics/retrieval-and-index-lifecycle.md]]
- [[.brain/planning/epics/context-and-session-workflow.md]]
- [[.brain/planning/epics/task-context-assembly.md]]
- [[.brain/planning/epics/structural-repo-context.md]]
- [[.brain/planning/epics/live-work-context.md]]
- [[.brain/resources/references/retrieval-and-indexing.md]]
- [[.brain/resources/references/architecture-and-code-map.md]]
- [[.brain/resources/references/skills-and-context-engineering.md]]

## Raw Notes

Direction locked from the critique:

1. The real upgrade area is context assembly quality, not more durable memory.
2. The first wave should focus on three practical wins:
   - task context assembly
   - structural repo awareness
   - live work awareness
3. The first wave should stay understandable. Brain should not become an opaque context platform.
4. These items are explicitly deferred until later evidence justifies them:
   - relationship graphs as a first-class platform layer
   - deep language-specific code intelligence
   - incremental derived-state refresh as a headline initiative
   - a new dedicated context-packet command if existing command surfaces can carry the workflow

North-star statement:

> Brain assembles transparent, task-specific local context from canonical markdown, lightweight structural repo understanding, and live workspace signals while keeping all durable truth human-readable and version-controlled.
