# Planning Storage and Modes

## Status

Approved high-level modes. Local compatibility is the first implementation.
Hybrid synchronization details remain open.

Migration source modes are limited to `local`, `github`, and `hybrid`
only. During standalone Plan migration, “GitHub hybrid” is descriptive wording
for the canonical `hybrid` value and means split local/GitHub ownership; it is not
a fourth enum value. Linear is not a Brain Planning source mode, integration,
configuration surface, or target adapter.

## Local

Initial Brain Planning retains human-readable repository-local `.plan/` storage
where practical:

```text
.plan/
  PROJECT.md
  ROADMAP.md
  brainstorms/
  ideas/
  archive/
  specs/
  .meta/
```

This boundary preserves existing workspaces, permits coexistence with the Plan
CLI, avoids mass filesystem churn, and keeps a clear domain store. Planning data
does not move under `.brain/` during initial migration.

Brain Core owns project identity and module enablement. Planning owns `.plan/`
schema and adapters. Brain's normal knowledge index must not automatically ingest
`.plan/`; the enabled Planning context/search providers mediate selection,
permissions, provenance, and budgets.

## Cloud

Cloud Planning may be entirely Brain Cloud-native:

- no `.plan/` directory required
- Brain Cloud project/user identity and authorization
- revisioned brainstorm, spec, roadmap, approval, and execution records
- unified audit/events/APIs and future Brain Cloud UI
- agent and ChatGPT access through Brain Cloud capability discovery

Cloud Planning is absent when the project/account lacks the `planning`
capability.

## Hybrid

Ownership is explicit by Planning layer, conceptually:

```yaml
planning:
  ownership:
    project: local
    brainstorms: cloud
    roadmap: cloud
    specs: local
    execution: cloud
```

Likely flow:

1. Brainstorm through Brain Cloud or ChatGPT.
2. Compile relevant Brain/Hive context.
3. Refine, challenge, and approve a spec.
4. Generate a repository-local `.plan/specs/` change.
5. Open a planning artifact PR.
6. Track execution in Brain Cloud.
7. Propose Brain memory after verification.

Planning sync is a separate protocol from Brain context-memory sync. It needs
revision IDs, ownership rules, conflict detection, offline behavior, recovery,
idempotency, and audit. Phase 9 owns that design.

## Disablement and Migration

Disabling Planning stops commands/providers/jobs and creates nothing new. It does
not delete `.plan/` or cloud records. Re-enabling validates schema/migrations
before activation.

Existing workspaces are detected without mutation, inventoried, and upgraded
idempotently after preview/confirmation where data changes are required. Unknown
future schema versions must fail closed with a compatible-version diagnostic.

## Metadata

Current standalone Plan `.plan/.meta/` may include Linear JSON because its v3
adoption path creates that file even for local workspaces. Brain Planning must not
track or import it. A legacy workspace configured for Linear fails with
standalone Plan migration guidance before Brain Planning enablement.

Target metadata must separate:

- domain schema/revision
- local migration history
- guided/runtime session state
- optional integration mappings
- sync/reconciliation state

GitHub metadata remains transitional. Tool-managed JSON stays non-authoritative
when a human-readable artifact is canonical.
