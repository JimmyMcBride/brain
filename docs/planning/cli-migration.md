---
updated: "2026-07-31T19:59:02Z"
---
# Planning CLI Migration

## Status

Approved staged compatibility direction. Phase 3's bounded local native surface
is merged. Phase 4's approved
`.plan/specs/plan-cli-compatibility.md` defines the shared-package boundary,
mapped local workflow, full command disposition, compatibility contract, and
warning policy. Phase 4 now has a versioned conformance manifest pinned to
standalone Plan revision `53ebd96`; PR #46 captured root-command disposition
and initial `status` baselines, PR #47 normalized Brain's Go module path, and PR
#48 exposed the storage-neutral domain. The current slice captures aggregate
status, project/spec checks, and roadmap read/replace behavior; exposes
`planning/application` and `planning/local`; and migrates `brain plan status`,
`brain plan check`, and `brain plan roadmap`. Brain roadmap writes intentionally
add explicit confirmation, audit events, atomic replacement, and unchanged
reruns.

## Command Families

Target conceptual family:

```text
brain modules grant official.planning planning.read planning.brainstorm planning.roadmap project.context.read --project .
brain modules enable official.planning --project .
brain plan ...
```

Existing family:

```text
plan ...
```

`brain plan` is preferred over flattening Planning commands into Brain Core. Core
owns root flags, module enablement checks, help, command collisions, and output
conventions. Planning registers its namespaced command tree through a controlled
API.

## Stages

1. Current: standalone `plan` remains fully functional; Brain adds architecture
   and later module infrastructure without behavior changes.
2. Shared domain/application services: Brain owns importable local status,
   check, roadmap, brainstorm list/show/start, and spec list/show behavior.
   Standalone `plan` remains the pinned compatibility source until its mapped
   local families switch to these packages.
3. Native primary command: `brain plan` becomes primary; `plan` is a compatibility
   wrapper and emits documented warnings.
4. Distribution deprecation: workspace migration is stable; separate Plan release
   timing and support window are published.

## Compatibility Rules

- no existing script breaks before mapping/migration support exists
- wrapper and native command call shared implementation
- flags, exit codes, JSON output, confirmation, and idempotency receive
  compatibility tests
- module-disabled errors distinguish “not enabled” from invalid workspace/data
- local `.plan/` continues working with standalone and native commands during the
  supported overlap
- warnings go to stderr and do not corrupt JSON stdout
- removed Linear behavior gets an explicit unsupported-configuration error
- legacy epic/story commands remain only as long as migration/read compatibility
  requires

## Candidate Mapping

| Existing | Target | Direction |
| --- | --- | --- |
| `plan doctor` | `brain plan doctor` or module health detail | Evaluate with Phase 3 UX |
| `plan source` | `brain plan storage`/configuration | Redesign ownership terminology |
| `plan brainstorm ...` | `brain plan brainstorm ...` | Preserve |
| `plan discuss ...` | `brain plan brainstorm assess/promote` or retained group | Evaluate discoverability |
| `plan spec ...` | `brain plan spec ...` | Preserve |
| `plan roadmap ...` | `brain plan roadmap ...` | Preserve |
| `plan github ...` | companion GitHub namespace/integration settings | Transition |
| `plan epic/story ...` | compatibility commands/import tools | Retire after migration |
| `plan skills ...` | Brain module/agent-tool installation | Redesign |

Do not finalize mapping before testing help/navigation and machine-output
compatibility against the full command inventory.
