---
created_at: "2026-07-01T00:00:00Z"
project: fixture
slug: draft-spec
status: draft
title: Draft Spec
type: spec
updated_at: "2026-07-01T00:00:00Z"
---

# Draft Spec

## Why

Exercise explicit approval.

## Problem

Draft specs cannot start execution.

## Goals

- Approve a complete local spec.

## Non-Goals

- Starting execution automatically.

## Constraints

- Require explicit confirmation in Brain.

## Solution Shape

Apply a guarded lifecycle transition.

## Flows

1. Review the draft.
2. Approve it.

## Data / Interfaces

- Spec status metadata.

## Risks / Open Questions

- Host authorization differs.

## Rollout

- Verify both hosts.

## Verification

- Run lifecycle tests.
