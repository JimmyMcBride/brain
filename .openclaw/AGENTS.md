# Openclaw Wrapper

<!-- brain:begin agent-wrapper-openclaw -->
This `openclaw` wrapper delegates to the root project contract.

## Required Reads

- `../AGENTS.md`
- `../.brain/policy.yaml`
- `../.brain/context/overview.md`
- `../.brain/context/architecture.md`
- `../.brain/context/workflows.md`
- `../.brain/context/memory-policy.md`

## Required Behavior

- Treat `../AGENTS.md` as the canonical project contract.
- If no validated session is active, run `brain session start --task "<task>"`.
- If a session is already active, run `brain session validate` before substantial work.
- Use the `brain` skill and `brain` CLI when project memory or vault context matters.
- Use `brain session run -- <command>` for required verification commands.
- Finish with `brain session finish` and mention relevant note updates in the final response.
<!-- brain:end agent-wrapper-openclaw -->

## Local Notes

Add repo-specific notes here. `brain context refresh` preserves content outside managed blocks.
