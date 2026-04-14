---
created: "2026-04-13T22:00:56Z"
epic: live-work-context
project: brain
status: approved
title: Live Work Context Spec
type: spec
updated: "2026-04-13T23:05:03Z"
---
# Live Work Context Spec

Created: 2026-04-13T22:00:56Z

## Why

The right context is not only about what is true in the repo. It is also about what is active now. Brain already has sessions, git-aware closeout, verification history, and structural repo context, but it does not yet use those signals deeply enough to improve task focus.

## Problem

Brain currently has limited awareness of the specific work in progress beyond task text and note ranking. That leaves it weaker than it should be at choosing nearby tests, touched files, relevant workflow guidance, and structural boundaries for the current task.

## Goals

- Add practical live-work signals that improve task focus without changing Brain's canonical truth model.
- Compute live-work context on demand from current session, git, verification, policy, and structural signals.
- Improve task-context assembly and retrieval quality with these signals while keeping the result explainable.
- Stop short of introducing a broad relationship-graph platform in the first wave.
- Keep the first wave independent from project-planning proximity so the planning system can later move behind a plugin boundary if needed.

## Non-Goals

- Building a repo-wide relationship graph as the first implementation.
- Storing git history or session telemetry as new durable memory by default.
- Replacing explicit policy or workflow docs with silent heuristics.
- Treating live-work context as stable truth when it is only current-state signal.
- Adding persistent live-work caches or a live-work status command in the first wave.
- Including planning proximity in the first wave.

## Requirements

- Add `brain context live` as the direct inspection command for live-work context.
- Support `brain context live --task "<task>"` and `brain context live` using the active session task.
- Support `brain context live --explain` for richer rationale and missing-signal reporting.
- Compute live-work context on demand rather than persisting it to SQLite or the active session file.
- Derive live-work context from active session state when available.
- Include changed-file awareness as a first-class signal using current git state and session git baseline when present.
- Include touched structural boundaries by consuming the structural repo context layer.
- Include nearby tests derived from changed files and touched boundaries.
- Include recent recorded session commands with exit codes and verification-profile satisfaction.
- Include workflow or policy hints only on strong-match conditions.
- Keep the use of live-work context inspectable so a user can understand why Brain emphasized certain files, tests, or workflow guidance.

## UX / Flows

Active-session flow:
1. User starts or validates a session for a task.
2. User runs `brain context live`.
3. Brain derives live-work context from the active task, current repo state, recorded command runs, and structural boundaries.
4. Brain returns a compact live-work view with any ambiguity notes.

Explicit-task fallback flow:
1. User runs `brain context live --task "tighten auth flow"` without an active session.
2. Brain falls back to explicit task text plus whatever git/worktree and structural signals exist.
3. Brain returns live-work context with weaker-signal ambiguities when session-specific data is missing.

Explain flow:
1. User runs `brain context live --explain`.
2. Brain returns the normal live-work view plus why the selected signals matter and which live signals are currently missing.

Missing-task flow:
1. User runs `brain context live` with no active session and no `--task`.
2. Brain returns a clear error explaining that a task or active session is required.

## Data / Interfaces

Public command surface:
- `brain context live`
- `brain context live --task "<task>"`
- `brain context live --explain`

Resolution rules:
- `--task` wins when provided.
- Otherwise use the active session task.
- If neither exists, fail clearly.

Default human output sections:
- `## Task`
- `## Session`
- `## Changed Files`
- `## Touched Boundaries`
- `## Nearby Tests`
- `## Verification`
- `## Policy Hints` only when non-empty
- `## Ambiguities` only when non-empty

Explain-mode-only human output sections:
- `## Why These Signals Matter`
- `## Missing Live Signals`

Stable JSON contract:

```json
{
  "task": {
    "text": "tighten auth flow",
    "source": "flag"
  },
  "session": {
    "active": true,
    "id": "1776119603034163976",
    "started_at": "2026-04-13T22:33:23Z"
  },
  "worktree": {
    "git_available": true,
    "baseline_head": "abc123",
    "current_head": "def456",
    "changed_files": [],
    "touched_boundaries": []
  },
  "nearby_tests": [],
  "verification": {
    "recent_commands": [],
    "profiles": []
  },
  "policy_hints": [],
  "ambiguities": []
}
```

Changed-file item shape:

```json
{
  "path": "internal/search/search.go",
  "status": "modified",
  "source": "worktree",
  "why": "changed since session baseline"
}
```

Touched-boundary item shape:

```json
{
  "path": "internal/search/",
  "label": "internal/search",
  "role": "library",
  "why": "contains changed files"
}
```

Nearby-test item shape:

```json
{
  "path": "internal/search/search_test.go",
  "relation": "same_dir",
  "why": "test surface near changed code"
}
```

Verification-command item shape:

```json
{
  "command": "go test ./...",
  "exit_code": 0,
  "started_at": "2026-04-13T22:35:00Z",
  "ended_at": "2026-04-13T22:35:04Z"
}
```

Verification-profile item shape:

```json
{
  "name": "tests",
  "satisfied": true,
  "matched_command": "go test ./..."
}
```

Policy-hint item shape:

```json
{
  "source": ".brain/context/workflows.md",
  "label": "Verification workflow",
  "excerpt": "Run required verification commands through `brain session run -- <command>`.",
  "why": "repo changes detected but required verification is still missing"
}
```

## Signal Model

First-wave signal families:
- `task`
- `session`
- `worktree.changed_files`
- `worktree.touched_boundaries`
- `nearby_tests`
- `verification.recent_commands`
- `verification.profiles`
- `policy_hints`
- `ambiguities`

Changed-file detection:
- If no active session exists, use current git state when available.
- If a session exists, union:
  - paths changed between `GitBaseline.Head` and current `HEAD`
  - current worktree status paths
- Normalize to repo-relative slash paths.

Changed-file status mapping:
- `M` -> `modified`
- `A` -> `added`
- `D` -> `deleted`
- `R` -> `renamed`
- otherwise `unknown`

Touched-boundary derivation:
- Resolve changed files against the structural repo context layer.
- Include the nearest structural boundaries that contain changed files.

Nearby-test derivation:
1. Include changed test files first.
2. Otherwise include sibling test files in the same directory matching:
   - `*_test.go`
   - `*.test.*`
   - `*.spec.*`
3. If none exist, include test surfaces from the same touched boundary via structural repo context.
4. Limit default human output to `5` nearby tests.
5. Return full arrays in JSON.

Verification signals:
- Include recent recorded session commands with both pass and fail exit codes.
- In default human output, show the latest `5` recorded commands.
- In JSON output, include all recorded commands from the current active session.
- Evaluate policy closeout verification profiles against recorded session command runs.
- Include `matched_command` when a profile is satisfied.

Policy-hint selection:
Only include policy or workflow hints when one of these strong-match conditions is true:
- no active session exists and `--task` fallback is being used
- repo changes exist and one or more required verification profiles are unsatisfied
- repo changes exist and no qualifying durable note updates have been recorded since the session baseline

Policy-hint sources:
- `AGENTS.md`
- `.brain/context/workflows.md`
- `.brain/context/memory-policy.md`
- `.brain/policy.yaml`

## Ambiguity Rules

Add ambiguity notes when:
- no active session exists and Brain is using only `--task`
- git is unavailable
- there are no changed files
- no nearby tests can be derived
- no verification commands have been recorded yet
- structural context is unavailable so touched boundaries cannot be computed

There is no top-level confidence field in `context live` v1. Ambiguities are the explicit expression of weak or missing signals.

## Storage / Lifecycle

- Compute live-work context on demand each time `brain context live` or `brain context assemble` needs it.
- Do not persist a live-work snapshot in the session file.
- Do not add live-work tables to SQLite.
- Introduce a dedicated live-work subsystem instead of burying the logic inside `session.Manager`.
- Reuse existing session state, git helpers, policy loading, and structural context as inputs.

## Task Context Integration

This epic populates the reserved `live_work` group in Task Context Assembly.

Mapping rules:
- changed files, touched boundaries, nearby tests, verification profiles, and policy hints become candidate `live_work` items
- each mapped packet item must provide `source`, `label`, `kind`, `excerpt`, and `why`
- this epic does not change `brain search`
- this epic makes live-work signals available to `context assemble`; task-packet weighting remains owned by Task Context Assembly

## Risks / Open Questions

- Are the first-wave changed-file heuristics expressive enough for non-git or weakly structured repos?
- Is `context live` enough as a direct inspection surface, or will later debugging need a richer status/query layer?
- Are strong-match-only policy hints strict enough, or will some workflows still need more always-visible guardrails?

## Rollout

1. Add the on-demand live-work subsystem and `brain context live` command.
2. Implement active task, changed-file, touched-boundary, and nearby-test signals first.
3. Add verification and strong-match policy hints next.
4. Wire live-work signals into the `live_work` group for `context assemble`.

## Story Breakdown

- [ ] Add the `brain context live` command and stable live-work JSON contract.
- [ ] Implement on-demand changed-file, touched-boundary, and nearby-test signals.
- [ ] Add verification and strong-match policy hints, plus ambiguity reporting.

## Resources

- [[.brain/brainstorms/context-engine-v2.md]]
- [[.brain/planning/specs/context-and-session-workflow.md]]
- [[.brain/planning/specs/task-context-assembly.md]]
- [[.brain/planning/specs/structural-repo-context.md]]
- [[.brain/resources/references/agent-workflow.md]]

## Notes

This epic is about improving focus around real work in progress, not building a general live graph of the repo. Planning proximity is deliberately deferred from the first wave so the live-work model degrades cleanly even if planning later moves behind a plugin boundary.
