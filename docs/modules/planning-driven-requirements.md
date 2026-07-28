# Planning-Driven Module Requirements

## Purpose

This matrix separates general Brain platform needs from Planning domain behavior.
“Existing” means current implementation evidence, not a stable public API.

| Requirement | Needed by Planning | Generalizable | Owner | Existing support | Proposed contract |
| --- | ---: | ---: | --- | --- | --- |
| Project identity | Yes | Yes | Core | Project root/path only | Stable project reference shared by all modules |
| Module discovery | Yes | Yes | Core | None | Compiled descriptor registry |
| Enable/disable state | Yes | Yes | Core | None | Explicit project-scoped state; disabled by default |
| Compatibility check | Yes | Yes | Core | Version command only | Brain API range validation |
| Lifecycle/migrations | Yes | Yes | Core | Project context migrations | Idempotent module lifecycle and schema version |
| Namespaced commands | Yes | Yes | Core | Central Cobra wiring | Controlled command registrar and collision rules |
| Module configuration | Yes | Yes | Core | Global embedding/output config | Namespaced global/project/cloud config schemas |
| Secret references | Integrations | Yes | Core | None | Opaque scoped references, never raw manifest values |
| Permissions | Yes | Yes | Core | Workflow policy only | Declare, grant, check, audit |
| Typed events | Yes | Yes | Core | Session/history records only | Versioned event registry and request identity |
| Audit primitives | Yes | Yes | Core | History/session logs | Attributable action records |
| Context provider | Yes | Yes | Core | Compilers consume concrete sources | Budgeted provider contract with provenance |
| Search provider | Yes | Yes | Core | Local Markdown/SQLite search | Authorized provider and merged ranking contract |
| Memory proposals | Yes | Yes | Core | Distill/promotion proposals | Review-first proposal API |
| Agent tools | Yes | Yes | Core | Skills only | Typed schema, permission, mutation, confirmation |
| Background jobs | Cloud | Yes | Core | None | Deferred retry/idempotency registry |
| Cloud API registration | Cloud | Yes | Core/Cloud | None | Core-owned authenticated versioned routing |
| Web surfaces | Future | Yes | Core/Cloud | None | Deferred trusted registration |
| Local module storage | Yes | Yes | Core | Brain path helpers only | Scoped storage adapter; Planning can retain `.plan/` |
| Cloud module storage | Yes | Yes | Core/Cloud | None | Revisioned project/tenant-scoped storage |
| Hybrid ownership | Yes | Partly | Planning | None in Brain | Planning ownership policy over Core storage clients |
| Brainstorms/refinement/challenge | Yes | No | Planning | Plan implementation | Planning domain/workflow |
| Maturity/readiness/checklists | Yes | No | Planning | Plan implementation | Planning assessment engine |
| Specs/initiatives/roadmaps | Yes | No | Planning | Plan implementation | Planning domain repositories |
| Approvals/execution queues/slices | Yes | No | Planning | Partial Plan behavior | Planning workflow and policy |
| Planning source references | Yes | Pattern general | Planning over Core | Brain provenance exists in packets | Planning schema using Core source references |
| Completion memory proposals | Yes | Pattern general | Planning over Core | Distill proposals only | Planning proposal provider |
| GitHub publication/collaboration | Transitional | Provider pattern | Companion adapter | Plan GitHub-specific client/backend | Planning integration contracts; no GitHub types in domain |
| Linear integration | No | No | Outside Brain Planning | Historical Plan source mode | Refuse import; standalone Plan cleanup only |
| External process protocol | Future | Yes | Core | None | Phase 11 after official contracts prove stable |

## Phase 1 Minimum

Registration, enablement, namespaced configuration, permissions, lifecycle hooks,
capability registration, health, tests, and one trivial reference module. Context,
search, memory proposals, tools, jobs, APIs, and web contracts should be added only
when a concrete implementation phase needs them.
