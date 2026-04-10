# Claude Wrapper

<!-- brain:begin agent-wrapper-claude -->
This `claude` wrapper delegates to the root project contract.

## Required Reads

- `../AGENTS.md`
- `../.brain/context/overview.md`
- `../.brain/context/architecture.md`
- `../.brain/context/workflows.md`
- `../.brain/context/memory-policy.md`

## Required Behavior

- Treat `../AGENTS.md` as the canonical project contract.
- Use the `brain` skill and `brain` CLI when project memory or vault context matters.
- Capture durable context changes and mention them in the final response.
<!-- brain:end agent-wrapper-claude -->

## Local Notes

Add repo-specific notes here. `brain context refresh` preserves content outside managed blocks.
