---
created: "2026-04-16T02:07:44Z"
epic: derived-doc-capsules-and-drift-audit
project: brain
spec: derived-doc-capsules-and-drift-audit
status: todo
title: Teach Docs And Brain Skill About Capsules And Drift Audit
type: story
updated: "2026-04-16T05:39:00Z"
---
# Teach Docs And Brain Skill About Capsules And Drift Audit

Created: 2026-04-16T02:07:44Z

## Description

Update user and agent guidance so capsules are understood as derived, auditable compiler inputs rather than an always-injected rule-pack model.

## Acceptance Criteria

- [ ] `docs/usage.md`, `docs/skills.md`, or other relevant user docs explain when Brain uses capsules, how to inspect drift, and why full docs remain canonical.
- [ ] `skills/brain/SKILL.md` teaches capsule-backed compile and the available drift-inspection path without telling agents to depend on a perpetual injected rules layer.
- [ ] The guidance makes clear that capsules are a narrow third-wave optimization that should only follow budgets and reuse when remaining document-cost pressure justifies them.
- [ ] The guidance explains `off|auto|on`, makes clear that capsules ship default `off`, and states that `auto` is a telemetry-gated per-compile decision rather than a hidden permanent self-enable path.
- [ ] The wording explicitly rejects a Cursor-style always-injected `.mdc` model as the primary Brain architecture for this feature.

## Resources

- [[.brain/planning/specs/derived-doc-capsules-and-drift-audit.md]]
- [[docs/usage.md]]
- [[docs/skills.md]]
- [[skills/brain/SKILL.md]]

## Notes
