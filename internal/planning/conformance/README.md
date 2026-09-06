# Planning Conformance Manifests

This package freezes observable standalone Plan behavior before command families
move into Brain-owned shared implementation.

## Phase 4 local baseline

`testdata/manifest.yaml` owns:

- the exact standalone Plan repository and commit used as the baseline
- every standalone root command family's migration disposition
- the native mapping for Phase 4 local commands
- normalization rules for paths, line endings, and RFC3339 timestamps
- captured command cases and their output, filesystem, and rerun contracts
- native Brain arguments and any explicit host-policy differences

Golden output is captured manually from a standalone binary built at the pinned
revision. Each fixture is copied into an independent project root named
`fixture`, the command is invoked with `--project <PROJECT>`, and the absolute
project path is replaced with the manifest's `<PROJECT>` token. Line endings
become LF, paths use `/`, and volatile RFC3339 timestamps become `<TIMESTAMP>`
before comparison.

This first slice embeds and validates the manifest, fixtures, and goldens. It
does not fetch, build, or execute the pinned standalone repository during
`go test`; PR verification records the explicit baseline replay separately.

Human output starts with structural comparison. Exit codes, filesystem effects,
and rerun behavior are exact. JSON-producing guided and promotion cases use
small semantic projections of the pinned payload so host command hints and
other intentional differences stay explicit without weakening the workflow
decision, artifact identity, or mutation-action checks.

The initial cases freeze `plan status` against a compatible schema-v3 workspace,
an empty project, and a future-schema workspace. The next cases freeze local
project/spec checks and roadmap reads/replacements. Exact file goldens list each
changed relative path, followed by `---` and complete normalized content;
structural file goldens list the stable sections and metadata under comparison.
Roadmap writes record Brain's required confirmation and idempotent `unchanged`
rerun as intentional host-policy differences from the standalone baseline. The
guided fixture also freezes idea capture, session listing, current guide packet,
local maturity assessment, and direct-spec promotion preview. Native tests add
confirmed mutation/audit coverage and prove direct promotion creates no epic or
persisted story intermediate. Later Phase 4 slices add cases for their command
families before moving implementation.

## Phase 5 GitHub baseline

`testdata/github-v1/manifest.yaml` separately pins standalone Plan `v0.1.29` at
commit `898b2a4c470350c0e9115302c99c6ad27bdead9c`. Keeping a second manifest
preserves the older Phase 4 local baseline instead of silently rebasing it.

The GitHub manifest covers remote assessment, repair, and promotion; enablement,
adoption, reconciliation, and Project status; GitHub-aware check and guide
behavior; repository/planning-PR evidence; and only the shared evidence needed
by compatibility-only GitHub story creation. Each case names its standalone or
manager invocation, workspace and fake-provider state, normalized output, file
and `.plan/.meta/github.json` effects, provider-call transcript, remote
identities, confirmation policy, and identical-rerun outcome. Full text output
is exact where stable; JSON-heavy agent contracts use valid structural
projections while retaining action, identity, safety, and recovery fields.

The manifest explicitly records two target hardenings instead of treating them
as accidental drift: Brain requires preview and confirmation for GitHub
administration mutations that Plan `v0.1.29` applies immediately, and Brain must
suppress identical repair/adoption/Project-status writes that the baseline may
repeat. Partial publication failure retains completed remote identity and permits
manual fallback only when the typed result says so.

Required CI only loads embedded fixtures and injected-fake transcripts. It never
authenticates with GitHub, invokes `gh`, or creates a live repository resource.
