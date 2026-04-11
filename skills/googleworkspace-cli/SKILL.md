---
name: googleworkspace-cli
description: Use this skill when the user wants to work with Google Workspace from the terminal through the `gws` CLI, including Drive, Gmail, Calendar, Sheets, Docs, Chat, Admin, or cross-service workflows. Trigger whenever the user mentions `gws`, Google Workspace CLI, or asks to inspect, script, automate, query, or manage Google Workspace APIs from a shell or agent.
user-invocable: true
---

# Google Workspace CLI

Use this skill to operate the `gws` CLI safely and efficiently.

`gws` builds its command surface dynamically from Google's Discovery Service, so do not guess flags or payload shapes. Inspect the live schema before running non-trivial commands.

## First checks

1. Confirm the CLI is present with `command -v gws`.
2. If `gws` is missing, suggest one of these install paths:
   - download a release from `https://github.com/googleworkspace/cli/releases`
   - `npm install -g @googleworkspace/cli`
   - `brew install googleworkspace-cli`
3. Confirm authentication state before API work:
   - `gws auth setup` for first-time local setup
   - `gws auth login` for OAuth login
   - `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` or `GOOGLE_WORKSPACE_CLI_TOKEN` for headless or CI flows

Read [references/quickstart.md](./references/quickstart.md) when you need the install and auth paths.  
Read [references/service-map.md](./references/service-map.md) when you need to pick a Workspace service or explain the command model.

## Core workflow

1. Identify the target service, resource, and method.
2. Inspect the service first:
   ```bash
   gws <service> --help
   ```
3. Inspect the exact method before writing flags:
   ```bash
   gws schema <service>.<resource>.<method>
   ```
4. Build the command with:
   - `--params` for query and path parameters
   - `--json` for request bodies
   - `--output` for downloaded binary content
   - `--page-all` for paginated list calls
5. Prefer read-only operations first so you can confirm IDs, ranges, parents, and payload shape before mutation.
6. When the user wants automation, show the exact command and keep the output machine-readable unless they asked for a table.

## Command model

Use this shape for most commands:

```bash
gws <service> <resource> [sub-resource] <method> [flags]
```

Common examples:

```bash
gws drive files list --params '{"pageSize": 10}'
gws schema drive.files.list
gws sheets spreadsheets create --json '{"properties":{"title":"Q1 Budget"}}'
gws drive files list --params '{"pageSize": 100}' --page-all
```

## Safety rules

- Confirm with the user before running create, update, patch, delete, move, import, or sharing commands.
- Prefer `--dry-run` when the method supports it and the change could be destructive or noisy.
- Never print tokens, credentials, or secret file contents back to the user.
- Use the schema output instead of guessing JSON keys or required params.
- For large or sensitive responses, pipe through `jq` or save to a file instead of dumping everything inline.

## Shell rules

- In `zsh`, sheet ranges with `!` need double quotes:
  ```bash
  gws sheets spreadsheets values get --params '{"spreadsheetId":"ID","range":"Sheet1!A1:D10"}'
  ```
- Wrap JSON for `--params` and `--json` in single quotes so the shell does not touch inner double quotes:
  ```bash
  gws drive files list --params '{"pageSize": 5}'
  ```
- When chaining with `jq`, keep the raw `gws` output as JSON unless the user explicitly wants `table`, `csv`, or `yaml`.

## Output expectations

When helping the user, prefer this structure:

1. Briefly state what the command will do.
2. Show the exact `gws` command.
3. Call out any required IDs, scopes, files, or auth prerequisites.
4. Mention whether the command is read-only or mutating.

## When to escalate

- If the command fails because the required API is not enabled, point the user to `gws auth setup` or the relevant Google Cloud project configuration.
- If discovery or schema inspection shows a different resource or method name than expected, trust the live `gws schema` output over memory.
- If the user wants a repeated workflow across several Workspace services, consider composing a short shell script after validating each individual step.
