---
status: verified
title: Remaining Phase 5 GitHub specs
type: change
updated: "2026-10-26T14:32:42Z"
---
# Remaining Phase 5 GitHub Specs

The approved GitHub Adapter Transition now has five dependency-ordered child
specs for all remaining Phase 5 work:

1. coordination mapping persistence;
2. repository reconciliation;
3. execution workspace fields, items, drift, and status;
4. native Brain configuration, commands, policy, events, audit, and health;
5. cross-host conformance and coordinated Brain/Plan releases.

The split keeps each implementation slice independently reviewable and
revertible while preserving the umbrella contract. Every child spec documents
scope boundaries, current compatibility data, failure recovery, migration and
rollback policy, acceptance criteria, dependencies, and verification.

All five specs remain draft pending maintainer approval. Coordination mapping
persistence is the exact next slice.
