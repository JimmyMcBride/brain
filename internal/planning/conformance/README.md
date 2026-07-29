# Plan CLI Conformance Manifest

This package freezes observable standalone Plan behavior before Phase 4 moves a
command family into Brain-owned shared implementation.

`testdata/manifest.yaml` owns:

- the exact standalone Plan repository and commit used as the baseline
- every standalone root command family's migration disposition
- the native mapping for Phase 4 local commands
- normalization rules for paths and line endings
- captured command cases and their output, filesystem, and rerun contracts

Golden output is captured manually from a standalone binary built at the pinned
revision. Each fixture is copied into an independent project root named
`fixture`, the command is invoked with `--project <PROJECT>`, and the absolute
project path is replaced with the manifest's `<PROJECT>` token. Line endings
become LF and paths use `/` before comparison.

This first slice embeds and validates the manifest, fixtures, and goldens. It
does not fetch, build, or execute the pinned standalone repository during
`go test`; PR verification records the explicit baseline replay separately.

Human output starts with structural comparison. Exit codes, filesystem effects,
and rerun behavior are exact. JSON will be exact when JSON-producing command
families enter the manifest.

The initial cases freeze `plan status` against a compatible schema-v3 workspace,
an empty project, and a future-schema workspace. Later Phase 4 slices add cases
for their command families before moving implementation.
