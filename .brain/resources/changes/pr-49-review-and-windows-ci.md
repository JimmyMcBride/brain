---
source: session_distill
title: PR 49 Review And Windows CI
type: change
updated: "2026-08-03T07:23:13Z"
---
# PR 49 Review And Windows CI

PR #49 review follow-up moved roadmap preview resolution to `planning.read`; confirmed writes remain guarded inside `Service.UpdateRoadmap` by `planning.roadmap`. Editor commands now preserve quoted executable paths, and Planning check scope errors distinguish a missing spec slug from extra project arguments.

Windows conformance normalizes both actual output and embedded goldens to the manifest LF contract. Roadmap atomic-replacement mode preservation remains asserted on Unix; Windows skips the Unix permission-bit assertion because `os.FileMode.Perm` does not preserve `0600` there.

Verified with:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- Windows test-binary compilation for `./cmd`, `./internal/planning/conformance`, and `./planning/local`
