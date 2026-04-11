# Standards

<!-- brain:begin context-standards -->
Use this file for implementation and review expectations.

## Standards

- Keep code idiomatic Go with small, concrete abstractions.
- Prefer explicit tests for CLI behavior, indexing, retrieval, safety flows, and session enforcement.
- Record required verification through `brain session run -- ...` so finish-stage enforcement can validate it.

## CI

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
<!-- brain:end context-standards -->

## Local Notes

Add repo-specific notes here. `brain context refresh` preserves content outside managed blocks.
