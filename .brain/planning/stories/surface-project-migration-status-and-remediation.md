---
created: "2026-04-15T23:33:00Z"
epic: release-install-and-update-flow
project: brain
spec: release-install-and-update-flow
status: todo
title: Surface Project Migration Status And Remediation
type: story
updated: "2026-04-15T23:33:00Z"
---
# Surface Project Migration Status And Remediation

Created: 2026-04-15T23:33:00Z

## Description

Expose project migration health in the same places users already inspect Brain upgrade state so automatic changes stay inspectable and failures are actionable instead of mysterious.


## Acceptance Criteria

- [ ] `brain update` human and JSON output include project migration status and the applied migration ids when a current project uses Brain
- [ ] If the binary update succeeds but project migration fails, Brain returns an explicit partial-success error such as `binary updated, project migration incomplete`
- [ ] Lazy migration failures block project commands with clear remediation that points users to `brain doctor`, `brain context refresh --project .`, and `brain adopt --project .` as appropriate
- [ ] `brain doctor` reports whether project migrations are current, pending, or broken for the current repo


## Resources

- [[.brain/planning/specs/release-install-and-update-flow.md]]
- [[cmd/update.go]]
- [[cmd/doctor.go]]
- [[docs/usage.md]]

## Notes

- Keep status naming parallel to the skill refresh flow so update output stays easy to scan.
