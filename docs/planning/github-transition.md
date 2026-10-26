# GitHub Planning Transition

## Status

GitHub support is transitional and retained during migration. It is not the
permanent canonical Planning backend. Phase 5 implementation is governed by the
approved [GitHub Adapter Transition](../../.plan/specs/github-adapter-transition.md)
spec.

## Current Implementation

Standalone Plan currently supports the compatibility surface, while Brain now
provides the native optional adapter for the retained Phase 5 families:

- GitHub/hybrid source modes in `.plan/.meta/workspace.json`
- GitHub Discussions as collaboration sources
- maturity assessment, repair, and promotion drafts
- confirmed initiative/spec issue creation
- labels, milestones, parent/sub-issue and blocked-by relationships
- GitHub Project creation/connection and execution status
- issue adoption and local mappings in `.plan/.meta/github.json`
- story readiness/reconciliation after merged planning changes

Brain configuration is explicit and independently disableable:

```text
brain plan github enable|disable
brain plan brainstorm assess|repair|promote --discussion <number-or-url>
brain plan github adopt|reconcile
brain plan github project status
brain plan guide show --discussion <number-or-url> --stage <stage>
```

The module migrates config v1 to v2 with GitHub disabled. Enabling preflights
the authenticated `gh` CLI and current repository before initializing adapter
state. Disabling retains artifacts and mappings and makes every GitHub command
return a typed adapter-disabled diagnostic without provider work. Local Planning
health remains independent when GitHub authentication or access degrades.

Standalone implementation remains concentrated in `internal/planning/github_*`,
`collaboration*`, `cmd/github.go`, and `cmd/discuss.go`. Brain keeps provider
DTOs and transport in `planning/github`; `planning/application` exposes only
provider-neutral collaboration, publication, repository-evidence, mapping, and
execution contracts, and `cmd/plan_github.go` performs host mapping.

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
