---
title: "Testing And Operations"
type: "reference"
created: "2026-04-11T00:00:00Z"
updated: "2026-04-11T00:00:00Z"
source: "migrated_project_memory"
---
# Testing And Operations

## Core Commands

```sh
go test ./...
go build ./...
go vet ./...
go run . doctor --project .
go run . search --project . "context"
```

## Verification Guidance

- Run tests and build checks for meaningful CLI or service changes.
- Exercise targeted search flows when retrieval changes.
- Refresh project context when the contract or workflow changes.
- Keep the repo buildable and deterministic on Linux.

## Release Hygiene

- `brain version` and `brain update` are part of the public surface.
- Update docs and project memory when install or release behavior changes.
- Keep generated tracked files intentional and avoid polluting git with local runtime artifacts.
