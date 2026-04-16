## Summary
- describe the user-facing change in release-note language
- keep this focused on shipped behavior, not implementation mechanics

## Release Notes
<!-- This section is promoted into the published GitHub release body when this PR ships. -->
- list the 1-5 highest-signal user-visible changes
- write these as human-readable bullets that can ship directly as the release changelog
- if the PR is mainly a fix, say what was broken and what is now correct

## Verification
- go test ./...
- go build ./...

## Maintainer Notes
- if this PR changes Brain commands or agent-facing workflow guidance, update `skills/brain/SKILL.md`
- if `skills/brain/` changed, reinstall the local Brain skill for Codex and OpenClaw before closeout:
  - `brain skills install --scope local --agent codex --agent openclaw --project .`
