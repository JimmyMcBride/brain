---
source: session_distill
title: Retire Project Management Surface
type: change
updated: "2026-04-16T06:41:03Z"
---
Removed Brain's retired project-management surface from the shipped product.

This change deletes the dedicated CLI commands, internal packages, templates, checked-in note trees, and workspace scaffolding for that surface. Brain now stays focused on repo-local memory, compiled context, retrieval, session enforcement, and session-scoped distillation.

Follow-through in the same change removed the obsolete note-type weighting from search, stopped project migration detection from treating `.brain/project.yaml` as active workspace state, and rewrote the user-facing docs and Brain skill to match the narrower product surface.

Verified with:
- `go test ./...`
- `go build ./...`
