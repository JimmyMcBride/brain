# Brain Planning Vision

## Status

Approved direction; not implemented.

Brain Planning is an official optional Brain module. It turns rough work into
reviewed, executable planning artifacts while consuming Brain's context, memory,
retrieval, provenance, sessions, permissions, events, and audit capabilities.

The long-term product repositories are `brain`, `brain-cloud`, and
`brain-cloud-sdk-go`. Separate Plan Cloud, Plan Cloud SDK, identity, frontend, and
agent-gateway products are not part of the direction. The standalone
[`plan`](https://github.com/JimmyMcBride/plan) repository remains the reference
implementation and compatibility source throughout migration.

## Product Boundary

Planning owns brainstorms, refinement and challenge, maturity/readiness, specs,
initiatives, roadmaps, analysis/checklists, approvals, queues, runtime slices,
handoffs, execution history, Planning permissions/events/context, and
Planning-to-memory proposals.

Planning does not own another project identity, authentication model, context or
memory system, private Brain storage access, or automatic durable-memory writes.
It cannot require GitHub, Linear, Jira, Brain Cloud, or another external service.

## Optional but First-Class

Conceptual experience:

```text
brain modules enable planning --project .
brain plan brainstorm start --project . "Authentication overhaul"
brain plan spec show --project . authentication-overhaul
brain plan status --project .
```

Exact syntax is deferred to Phase 1/3 CLI review.

When disabled, Planning creates no storage or configuration, injects no context,
grants no permissions, and direct Planning commands explain how to enable it.
When enabled, it uses Brain identity and capabilities and works in supported
local, cloud, or hybrid modes.

## Knowledge Loop

```text
Brain context and memory
  -> Planning brainstorm
  -> refinement and challenge
  -> approved spec
  -> execution and verification
  -> Brain memory proposal
  -> reviewed durable memory
```

Planning can request architecture, decisions, constraints, conventions, current
state, similar work, Hive findings, contradictions, and repository state. Its
artifacts retain source provenance and freshness. Completion can propose updated
architecture, decisions, current state, reusable patterns, lessons, or Hive
memory; it cannot silently publish them.

## Preserve, Redesign, Transition, Retire

- Preserve/evolve: brainstorm, refinement, challenge, maturity, specs,
  initiatives, roadmaps, readiness, analysis/checklists, promotion previews,
  explicit confirmation, idempotent reconciliation, execution queues/slices, and
  human-readable local artifacts.
- Redesign through Brain: identity, configuration, context, auth, permissions,
  events, audit, cloud persistence, agent tools, lifecycle/health, memory
  proposals, cloud sync, and repository associations.
- Transitional: GitHub source mode and collaboration/tracking, standalone `plan`,
  and migration metadata.
- Retire: official Linear support, separate Plan Cloud/SDK/identity/registry,
  permanent GitHub canonicality, and legacy epic/story creation after read-only
  migration support is sufficient.

## Non-Goals

No complete module, cloud/hybrid sync, community loader, marketplace, frontend,
large code port, immediate CLI retirement, immediate Linear deletion, `.plan/`
relocation, sprint system, support desk, HR system, source hosting, CI, or deploy
platform is implemented by this architecture.
