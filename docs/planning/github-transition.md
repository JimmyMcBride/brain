# GitHub Planning Transition

## Status

GitHub support is transitional and retained during migration. It is not the
permanent canonical Planning backend. Phase 5 implementation is governed by the
approved [GitHub Adapter Transition](../../.plan/specs/github-adapter-transition.md)
spec.

## Current Implementation

Standalone Plan currently supports:

- GitHub/hybrid source modes in `.plan/.meta/workspace.json`
- GitHub Discussions as collaboration sources
- maturity assessment, repair, and promotion drafts
- confirmed initiative/spec issue creation
- labels, milestones, parent/sub-issue and blocked-by relationships
- GitHub Project creation/connection and execution status
- issue adoption and local mappings in `.plan/.meta/github.json`
- story readiness/reconciliation after merged planning changes

Implementation is concentrated in `internal/planning/github_*`,
`collaboration*`, `cmd/github.go`, `cmd/discuss.go`, and their tests. GitHub types
currently appear throughout the collaboration/application layer; Phase 5 must
push them behind adapters.

## Transition Roles

Short term:

- collaborative brainstorms
- planning issues/milestones/Projects
- planning artifact PRs and repository evidence
- temporary remote planning before Cloud Planning

Long term companion integration:

- repository and merge-event connection
- PR/branch/commit evidence
- planning artifact PR creation
- optional issue/execution-status mirroring
- optional Discussions collaboration

## Target Boundary

Planning domain uses generic capabilities for collaboration sources,
publication, repository changes, external work items, and execution status.
The approved application boundary covers collaboration sources, publication
targets, repository evidence, external mappings, and execution workspaces.
GitHub-specific node IDs, issue/project types, labels, milestone fields, GraphQL
details, and `gh` execution stay in the optional `planning/github` adapter.

No generic contract should reproduce every GitHub feature. Extract only behavior
used by local Planning plus a retained adapter.

## Compatibility

- do not remove GitHub source modes before adapter-backed migration exists
- preserve stable slugs, source links, issue IDs, relationships, milestone and
  project mappings
- keep preview-first and explicit confirmation
- repeated apply/reconcile remains idempotent
- ambiguous identity fails instead of matching titles
- compatibility CLI scripts keep working through shared implementation
- existing `.plan/.meta/github.json` is read in place
- Phase 5 keeps the current metadata shape and reads it in place; no backfill
- GitHub source mode remains supported through Phase 9 replacement and a
  separate reviewed deprecation decision

## Phase 5 Exit

- no GitHub types/imports in Planning domain
- existing promotion/reconciliation tests pass through adapter contracts
- local Planning works with no GitHub dependency
- GitHub can be disabled independently
- transition/deprecation timing for GitHub source mode is documented
