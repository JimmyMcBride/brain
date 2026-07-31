# Planning CLI Migration

## Status

Approved staged compatibility direction. Phase 3's bounded local native surface
is merged. Phase 4's approved
`.plan/specs/plan-cli-compatibility.md` defines the shared-package boundary,
mapped local workflow, full command disposition, compatibility contract, and
warning policy. Phase 4 implementation has started with a versioned conformance
manifest pinned to standalone Plan revision `53ebd96`; PR #46 merged its root
command disposition and first compatible, missing-workspace, and future-schema
`status` baselines. PR #47 normalized Brain's Go module path. The current slice
exposes the storage-neutral domain at `github.com/JimmyMcBride/brain/planning`
with a locked export surface, stdlib-only dependency gate, and canonical
external-package compile test. CLI behavior remains unchanged.

## Command Families

Target conceptual family:

```text
brain modules grant official.planning planning.read planning.brainstorm project.context.read --project .
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
2. Shared domain/application services: implemented in Brain for local status,
   brainstorm list/show/start, and spec list/show; standalone `plan` remains the
   compatibility source until Phase 4 shares the implementation.
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
