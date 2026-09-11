---
status: verified
title: GitHub publication coordination decisions
type: change
updated: "2026-09-11T23:46:10Z"
---
# GitHub Publication Coordination Decisions

Phase 5 publication plans now retain two provider-neutral coordination choices:

- one optional shared delivery group, which maps to a GitHub milestone in the adapter;
- one explicit execution-workspace decision: create, connect, or skip.

Direct publication previews containing at least five specs must include a workspace decision before provider inspection. Adoption remains unchanged until coordination identity can be applied and persisted end to end. Plans order the group before artifacts and relationships and the workspace decision last. Existing stable identities classify as reuse or unchanged; conflicting, missing, foreign-provider, or malformed identities fail closed. Completed group and non-skipped workspace actions require stable provider evidence, including no-op reruns.

The GitHub adapter currently rejects group and workspace apply actions as unsupported before any write. The next bounded work is milestone inspection/apply, followed by GitHub Project inspection/apply and mapping persistence.
