---
status: verified
title: GitHub milestone publication
type: change
updated: "2026-09-12T02:41:00Z"
---
# GitHub Milestone Publication

The GitHub publication adapter now maps the provider-neutral shared group to a
milestone. Groups carry the canonical complete publication membership so a
missing issue attachment is a reviewed update instead of a hidden side effect.

Inspection resolves milestone identity from a reviewed reference, attached
issues, retained metadata, or an exact case-insensitive title match within a
bounded listing. Conflicting member milestones, duplicate title matches,
malformed identities, and listing limits fail closed.

Apply creates or revision-guards a milestone rename before artifact actions.
Issue creates and updates include the resolved milestone; otherwise unchanged
issues are attached with their own final revision check. Completed and partial
evidence retain only resolved or written identities. Lost milestone responses
recover through inspection without duplicate creation, and identical reruns
perform no provider writes.

Review follow-up made repeated issue evidence comparisons use the canonical
issue revision rather than raw REST and `gh issue list` transport structs. This
prevents nested milestone transport-only fields from producing false concurrent
change conflicts. Fake milestone listings now preserve the provider's array
shape, with direct coverage for title recovery and grouped issue updates.

The next bounded Phase 5 slice is GitHub Project inspection/apply for the
reviewed create/connect/skip workspace decision, followed by coordination
mapping persistence and native rollout wiring.
