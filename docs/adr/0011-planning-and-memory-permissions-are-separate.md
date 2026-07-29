# ADR 0011: Planning and Memory Permissions Remain Separate

- Status: Accepted
- Date: 2026-07-27

## Context

Planning users may be allowed to draft, approve, execute, or publish plans without
authority to revise durable Brain knowledge. Combining permissions creates
privilege escalation through workflow completion.

## Decision

Planning permissions and Brain memory permissions are independent. Planning
approval/admin does not imply `memory.propose`, `memory.approve`, or `memory.edit`.
Each operation is checked and audited separately.

## Consequences

Policies are more explicit and least-privilege. Workflows must handle denied
proposal or approval operations clearly.

## Alternatives Considered

- Bundle permissions: simpler UX, excessive authority.
- Let modules define Core grants: breaks Core control.
- Disable Planning-to-memory: loses reviewed knowledge loop.

## Migration Implications

Phase 1 permission registry must support module and Core namespaces without
implicit inheritance. Phase 7 tests cross-boundary denial.

## Follow-up Work

Specify grant storage, organization policy, and confirmation UX in the module
framework implementation.
