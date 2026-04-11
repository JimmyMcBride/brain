# Project Agent Contract

<!-- brain:begin agents-contract -->
Use this file as the canonical project contract for `brain`.

Read the linked context files before substantial work. Prefer the `brain` skill and `brain` CLI for project memory, retrieval, and durable context updates.

## Table Of Contents

- [Overview](./.brain/context/overview.md)
- [Architecture](./.brain/context/architecture.md)
- [Standards](./.brain/context/standards.md)
- [Workflows](./.brain/context/workflows.md)
- [Memory Policy](./.brain/context/memory-policy.md)
- [Current State](./.brain/context/current-state.md)
- [Policy](./.brain/policy.yaml)

## Human Docs

- [README.md](./README.md)
- [architecture.md](./docs/architecture.md)
- [project-architecture.md](./docs/project-architecture.md)
- [project-overview.md](./docs/project-overview.md)
- [project-workflows.md](./docs/project-workflows.md)
- [skills.md](./docs/skills.md)
- [usage.md](./docs/usage.md)
- [why.md](./docs/why.md)

## Required Workflow

1. If no validated session is active, run `brain session start --task "<task>"`.
2. If a session is already active, run `brain session validate` before substantial work.
3. Read this file and the linked context files needed for the task.
4. Retrieve project memory with `brain find brain` or `brain search "brain <task>"`.
5. Use `brain edit` for durable context updates to AGENTS.md, docs, or .brain notes.
6. Use `brain session run -- <command>` for required verification commands.
7. Finish with `brain session finish` so policy checks can enforce memory updates and required command runs.
<!-- brain:end agents-contract -->

## Local Notes

Add repo-specific notes here. `brain context refresh` preserves content outside managed blocks.
