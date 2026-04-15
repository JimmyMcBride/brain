---
brainstorm_status: active
created: "2026-04-15T02:28:28Z"
idea_count: 0
project: brain
title: Context Substrate Direction
type: brainstorm
updated: "2026-04-15T02:28:52Z"
---
# Brainstorm: Context Substrate Direction

Started: 2026-04-15T01:28:46Z

## Focus Question

How should Brain narrow itself around memory, retrieval, context assembly, and session hygiene so it becomes the system that answers one question well: what should the agent know right now?

## Ideas

- Re-center Brain around durable memory and context orchestration.
  Brain is strongest when it remembers what matters, retrieves the right local context, and assembles it into something an agent can act on.
- Treat planning and project management as external.
  If another tool or skill already owns planning, Brain should support it by providing context and durable memory, not by trying to become that tool.
- Keep Brain focused on substrate work, not workflow ownership.
  Brain should help agents think with the right local knowledge, while other tools own the actual domain workflows layered on top.
- Shrink integrations to a narrow model.
  If integrations exist at all, they should mainly add context roots, durable artifacts, or small amounts of additive guidance rather than taking over core behavior.
- Preserve the Brain skill as a universal helper.
  If an agent already has the Brain skill active, then Brain is already doing the most important cross-tool job: grounding unrelated work in repo-local memory and context.
- Prefer ingestion over reinvention.
  If a separate local project-management tool exists, the better direction is for Brain to read its durable artifacts when useful, rather than reimplement project management inside Brain.
- Keep the main product question simple.
  Brain should optimize for one core outcome: giving the agent the best available local knowledge for the current task.
- Reframe “plugins” as thin integrations, not a platform identity.
  A heavy plugin platform risks pulling Brain toward workflow sprawl, while a narrow integration surface keeps Brain aligned with memory and context.
- Make naming reflect the narrower identity.
  The current name suggests something broader and more cognitive. A future name should signal grounding, recall, context, retrieval, substrate, or orientation rather than planning or general intelligence.

## Direction So Far

- Brain should be re-centered around memory, retrieval, context assembly, and session hygiene.
- Planning should be treated as external or, at most, as a thin first-party integration built on top of Brain.
- Plugins should shrink into a narrow integration model rather than a full extension platform.
- The product should optimize around the question: `What should the agent know right now?`
- The Brain skill already gives other tools and skills natural support because it grounds their work in local context.
- The rename should follow this narrower product identity rather than the broader planning/plugin direction.

## Naming Direction

Naming criteria:
- should suggest memory, recall, context, grounding, indexing, or orientation
- should not imply broad project management or a general-purpose cognitive layer
- should feel like a substrate or support layer for agents
- should still fit a local-first developer tool

Possible directions to explore:
- recall-oriented names
- context-oriented names
- grounding/orientation names
- index/reference names
- substrate/foundation names

## Related

- Current repo docs: [[docs/usage.md]]
- Current repo docs: [[docs/skills.md]]
- Current repo docs: [[docs/architecture.md]]
- Current repo context: [[.brain/context/current-state.md]]

## Raw Notes

- If an agent already has the Brain skill installed and active, it is naturally going to use Brain for local memory, retrieval, and context while doing unrelated work with other skills.
- That weakens the case for Brain owning project planning directly.
- Planning and plugins may be overengineered if they try to make Brain responsible for jobs that other tools can already do while Brain supplies context underneath them.
- The strongest argument against removing planning from Brain is that local project-management artifacts may contain useful context.
- That still does not require Brain to own planning.
  It suggests Brain should be able to read or index durable artifacts created by external project-management tools.
- The clean product model is:
  - Brain = memory and context substrate
  - other tools = workflow owners
  - Brain helps agents use those tools with the right local knowledge
- If integrations stay in scope, they should probably be limited to things like:
  - additional context roots or declared files
  - additive skill guidance
  - maybe a few human-facing helper commands
- Integrations should not redefine search, session lifecycle, or the core meaning of Brain.
- The main product test should be:
  if a feature does not make Brain materially better at remembering, retrieving, or assembling the right context for an agent, it probably does not belong in core.
- Renaming should happen after the narrower identity is stable enough to describe clearly.
  The next step is likely a naming pass against this substrate-focused direction rather than immediate codebase renaming.
