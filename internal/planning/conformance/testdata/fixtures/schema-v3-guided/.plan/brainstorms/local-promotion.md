---
created_at: "2026-07-01T00:00:00Z"
project: fixture
slug: local-promotion
status: active
title: Local Promotion
type: brainstorm
updated_at: "2026-07-01T00:00:00Z"
---

# Brainstorm: Local Promotion

Started: 2026-07-01T00:00:00Z

## Vision

Promote a refined local brainstorm into a single canonical spec.

## Supporting Material

- [Planning migration](../../docs/planning/cli-migration.md)

## Focus Question

How should local promotion preserve safety?

## Desired Outcome

One reviewable spec without an epic intermediate.

## Constraints

- Keep promotion preview-first.
- Do not create GitHub issues.

## Refinement

### Problem

Local promotion lacks a shared maturity and mutation contract.

### User / Value

Agents get one canonical spec without guessing the next artifact.

### Appetite

One bounded compatibility slice.

### Remaining Open Questions

- Which host renders migration warnings?

### Candidate Approaches

- Assess maturity from stable brainstorm sections.
- Preview the exact spec mutation.

### Decision Snapshot

Promote directly into one spec after explicit review.

## Challenge

### Rabbit Holes

- Do not build remote collaboration.

### No-Gos

- No epic intermediate.
- No persisted execution slices.

### Assumptions

- The local workspace remains schema v3.

### Likely Overengineering

Building a generic workflow engine during compatibility migration.

### Simpler Alternative

Share only the local application and adapter seams required by captured cases.
