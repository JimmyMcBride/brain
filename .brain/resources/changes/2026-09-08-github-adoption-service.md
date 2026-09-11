---
status: verified
title: Phase 5 GitHub adoption service
type: change
updated: "2026-09-09T04:44:18Z"
verification: go test ./...; go test -race ./...; go vet ./...; go build ./...
---
The provider-neutral adoption service now previews explicit existing-object
candidates, revalidates reviewed intent and remote revisions, applies canonical
publication updates, persists complete mappings only after provider success,
and emits one audit event only when remote or mapping state changed.

GitHub resolves explicit issue candidates by canonical repository/number
identity before title or label discovery, preserves exact legacy
`.plan/.meta/github.json`, and supports update/reuse classifications with no-op
reruns. Provider, mapping, and audit failures retain partial evidence without
rollback.

Verification includes application authorization/confirmation/stale-plan gates,
ambiguity rejection, JSON round trips, partial-failure evidence, direct
unmanaged-issue adoption through the fake GitHub transport, and identical
reruns.

Next bounded Phase 5 slice: retained milestone/workspace decision planning,
before native command wiring and complete cross-host conformance.
