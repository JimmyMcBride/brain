---
status: verified
title: GitHub Project publication identity
type: change
updated: "2026-09-26T22:30:48Z"
---
# GitHub Project Publication Identity

The GitHub publication adapter now applies the provider-neutral reviewed
create, connect, or skip workspace decision to GitHub Project identity.

Inspection accepts a canonical Project node/number/URL reference or recovers a
create by finding exactly one case-insensitive title match already linked to the
target repository. Owner, URL, repository evidence, and bounded listings are
validated. Title alone is never sufficient, and ambiguous, malformed, or
incomplete evidence fails closed.

Apply re-inspects the complete reviewed plan, performs a final recovery check,
creates the Project with its repository association, and returns a revisioned
reference. Uncertain responses retain partial evidence; inspection recovers the
created Project so a rerun does not duplicate it. Connect and skip perform no
Project mutation, and identical reruns are write-free.

This slice owns Project identity only. Project field configuration, item
attachment, status, and drift stay behind the execution-workspace port because
the publication decision does not expose those mutations for review.

The next bounded Phase 5 slice persists milestone and workspace identities
through adoption and reconciliation.
