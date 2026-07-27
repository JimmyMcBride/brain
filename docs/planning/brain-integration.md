# Planning and Brain Integration

## Principle

Planning consumes Brain through formal module contracts. It does not import
private Core managers, read Brain SQLite directly, or treat `.brain/` files as an
undocumented API.

## Shaping Context

Planning may request architecture, decisions, constraints, conventions, current
state, related prior work, Hive findings, contradictions, repository state, and
session evidence. Core performs authorization, retrieval, budgeting, and final
provenance handling.

Planning-specific compilation can combine:

- the artifact being shaped
- linked initiative/roadmap state
- approved Brain context
- source references already attached to the artifact
- current repository/session signals
- bounded integration evidence

Each contribution records source identity, revision/freshness, visibility, module
ID, selection rationale, and purpose. Stale or unavailable sources are visible,
not silently dropped from the artifact's history.

## Events

Candidate Core events consumed by Planning:

- project initialized
- context updated
- memory revised
- session finished
- sync completed
- contradiction detected
- pull request merged

Candidate Planning events:

- brainstorm created/refined/challenged
- spec drafted/approved/reopened
- planning execution started/completed
- integration publish/reconcile completed
- memory proposal submitted/approved/rejected

Events are typed/versioned facts, not hidden callbacks. Event receipt does not
grant mutation permission. Retryable consumers require idempotency keys.

## Permissions

Planning domain permissions are distinct from Core memory permissions. Reading
Brain context requires a Core grant. Proposing memory requires `memory.propose`.
Approving or editing memory requires separate grants. A Planning administrator
cannot bypass them.

## Completion-to-Memory

After verified execution, Planning can draft a proposal containing:

- source spec/execution identity
- verified outcome and evidence
- proposed architecture, decision, current-state, lesson, or Hive update
- source provenance
- confidence and unresolved contradictions
- suggested durable target

Brain owns review, approval, editing, history, and final persistence. Default
Planning behavior never directly writes durable Brain memory.

## Agent Tools

Future tools expose typed Planning operations with stable input/output schemas,
permissions, mutation class, confirmation, and audit behavior. The CLI and agent
tools should call shared application services, not duplicate workflow logic.

## Failure Behavior

- disabled Planning: clear enablement diagnostic
- unavailable/stale Brain source: preserve reference and report status
- denied context/memory permission: return explicit authorization failure
- provider timeout: omit bounded contribution with diagnostic; preserve Core use
- proposal rejected: retain Planning outcome and review evidence without mutation
