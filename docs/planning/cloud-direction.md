# Brain Cloud Planning Direction

## Product Boundary

Planning uses Brain Cloud identity, projects, permissions, revisions, events,
audit, APIs, jobs, and frontend. There is no separate Plan Cloud, Plan identity,
Plan frontend, Plan gateway, or `plan-cloud-sdk-go`.

Capability discovery must permit absence:

```json
{
  "capabilities": [
    "projects",
    "context",
    "memory",
    "search",
    "hive",
    "modules",
    "planning"
  ]
}
```

A client must not call Planning endpoints when `planning` is absent.

## SDK

Planning belongs in `brain-cloud-sdk-go`, conceptually:

```text
client.Projects()
client.Context()
client.Memory()
client.Search()
client.Hive()
client.Modules()
client.Planning()
```

This document does not approve method or request schemas.

## API Areas

Potential versioned areas:

```text
/v1/planning/brainstorms
/v1/planning/specs
/v1/planning/initiatives
/v1/planning/roadmaps
/v1/planning/execution
/v1/planning/approvals
```

Brain Cloud Core owns routing, authentication, authorization, tenancy, rate and
resource policy, revision semantics, and audit. Planning owns domain validation
and workflows behind those controls.

## Cloud-Native Requirements

- project/user identity shared with Brain
- optimistic revisions and conflict reporting
- Planning-specific permission checks
- attributable events and audit
- idempotency for mutations/jobs
- capability negotiation across clients
- data export and deletion policy
- no `.plan/` requirement
- agent/tool access through the same service layer as web/API

## Hybrid Boundary

Hybrid Planning is Planning-domain synchronization, not Brain memory/context sync.
Phase 9 defines layer ownership, cloud-to-local artifact generation, PR evidence,
conflicts, offline behavior, recovery, and metadata.

## UI

Future unified Brain Cloud navigation may include brainstorms, spec editor,
roadmap, approvals, execution, Brain context panel, and Hive integration. No
frontend extension mechanism is implemented or secured by this direction.
