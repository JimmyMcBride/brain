---
title: GitHub Conformance and Release Gate
type: reference
updated: "2026-10-26T20:40:56Z"
---
Brains Phase 5 GitHub conformance gate derives native `brain plan` arguments from the pinned standalone Plan v0.1.29 manifest and replays every migrated family through the real command tree with copied workspaces and injected provider/git runners. The `github.reconcile` replay also proves the non-CLI repository-evidence family. Required CI never invokes `gh` or live GitHub.

Schema-v3 `github` and `hybrid` workspaces expose their retained `.plan/` documents through `planning/local` as a read-only compatibility mirror. Reads support native reconciliation and GitHub-extended checks; every local adapter mutation remains blocked by `Writable: false` and the provider adapter owns external mutations.

The Brain release workflow contract is test-locked to callable tag/SHA inputs, checkout of the requested SHA, exactly Linux/macOS/Windows on amd64/arm64, versioned archive naming, and SHA-256 checksum publication. After this implementation PR merges, Brain must publish and verify the stable tag before standalone Plan pins it in a separate PR. Do not mark the coordinated release spec or Phase 5 complete until both releases and their artifacts are verified.
