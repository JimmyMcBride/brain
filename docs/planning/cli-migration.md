---
updated: "2026-08-10T07:27:42Z"
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
#48 exposed the storage-neutral domain, and the fourth slice migrated aggregate
status, project/spec checks, and roadmap read/replace behavior. The fifth slice
captures local brainstorm, guided-session, and direct-promotion behavior and
migrates `brain plan brainstorm idea/refine/challenge/resume/sessions/switch/reopen/review/park/assess/promote/repair`
plus `brain plan guide current/show`. Brain mutations intentionally add explicit
confirmation, permission checks, audit events, atomic local writes, and unchanged
reruns. Local promotion writes canonical specs directly and never creates an epic
or persisted execution-slice intermediate. The sixth slice migrates `brain plan
spec edit/status/analyze/checklist/initiative/execute/handoff` with captured
standalone contracts, preview-first writes, explicit spec permissions, guarded
handoff rollback, and ephemeral execution slices.
Standalone Plan PR [`#87`](https://github.com/JimmyMcBride/plan/pull/87)
implements the seventh slice against Brain merge `c2c71279030f`: compatible
schema-v3 local families now call the shared application/local packages, while
the standalone host retains flags, prompts, rendering, GitHub/hybrid ownership,
and legacy repair fallbacks. Its Linux and Windows CI are green pending merge.

## Command Families

Target conceptual family:

```text
brain modules grant official.planning planning.read planning.brainstorm planning.roadmap planning.spec.edit planning.spec.approve planning.execute project.context.read --project .
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
   check, roadmap, brainstorm capture/refinement/challenge, guided sessions and
   packets, direct local promotion, and the complete canonical local spec
   edit/readiness/approval/execution/handoff workflow.
   Standalone `plan` remains the pinned compatibility source; PR `#87` switches
   its mapped local families to these packages and awaits merge.
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
