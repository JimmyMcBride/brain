---
created: "2026-04-15T23:34:00Z"
epic: release-install-and-update-flow
project: brain
spec: release-install-and-update-flow
status: todo
title: Teach Docs And The Brain Skill About Automatic Project Upgrades
type: story
updated: "2026-04-15T23:34:00Z"
---
# Teach Docs And The Brain Skill About Automatic Project Upgrades

Created: 2026-04-15T23:34:00Z

## Description

Update the user-facing guidance so Brain explains that project soft migrations happen automatically during upgrade or first later use, while still documenting the explicit fallback commands when users need to intervene.


## Acceptance Criteria

- [ ] `docs/usage.md`, `docs/skills.md`, and any affected workflow docs explain automatic project migration for the current `--project` during `brain update` and lazy migration in other repos
- [ ] `skills/brain/SKILL.md` teaches the new upgrade behavior and the remediation path when an automatic project migration fails
- [ ] Maintainer workflow notes cover validating project migrations from a branch-built binary in the same way local skill bundle changes are validated
- [ ] The release/install/update planning notes are refreshed in the same branch so the documented lifecycle matches the implementation


## Resources

- [[.brain/planning/specs/release-install-and-update-flow.md]]
- [[docs/usage.md]]
- [[docs/skills.md]]
- [[skills/brain/SKILL.md]]

## Notes

- Treat this as required follow-through, not optional polish. Automatic migrations change user trust in upgrade behavior, so the docs and Brain skill must explain it.
