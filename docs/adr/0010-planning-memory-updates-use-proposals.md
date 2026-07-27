# ADR 0010: Planning Memory Updates Use Brain Proposals

- Status: Accepted
- Date: 2026-07-27

## Context

Completed plans produce valuable decisions and lessons. Silent writes would blur
planning with durable memory, bypass review, and risk stale or incorrect context.

## Decision

Planning submits attributable Brain memory proposals after verified outcomes.
Brain owns review, approval, editing, history, and persistence. Direct writes are
not Planning's default.

## Consequences

Knowledge capture is deliberate and auditable. Users perform a review step.
Planning retains source/evidence even when a proposal is rejected.

## Alternatives Considered

- Automatic writes: convenient but unsafe and hard to audit.
- No knowledge loop: loses high-value outcomes.
- Store Planning summaries as memory directly: duplicates systems.

## Migration Implications

Phase 7 adds source references, proposal payloads, freshness, and permission tests.

## Follow-up Work

Define the general memory-proposal contract from Brain's current distill/promotion
evidence.
