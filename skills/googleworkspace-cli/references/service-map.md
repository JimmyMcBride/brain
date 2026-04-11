# Workspace Service Map

Use this file to pick the correct `gws` service before inspecting a method schema.

## Core services

- `drive`
  - Files, folders, permissions, shared drives, comments, revisions
  - Start with `gws drive --help`
- `gmail`
  - Mailboxes, messages, drafts, labels, threads
  - Start with `gws gmail --help`
- `calendar`
  - Calendars, events, attendees, ACLs, free/busy
  - Start with `gws calendar --help`
- `sheets`
  - Spreadsheets, values, batch updates
  - Start with `gws sheets --help`
- `docs`
  - Document read and write operations
  - Start with `gws docs --help`
- `slides`
  - Presentations and slide updates
  - Start with `gws slides --help`
- `chat`
  - Spaces, messages, memberships
  - Start with `gws chat --help`
- `tasks`
  - Task lists and tasks
  - Start with `gws tasks --help`
- `people`
  - Contacts and profile information
  - Start with `gws people --help`
- `admin-reports`
  - Workspace audit and usage reporting
  - Start with `gws admin-reports --help`

## Discovery workflow

For an unfamiliar service, always follow:

```bash
gws <service> --help
gws schema <service>.<resource>.<method>
```

Examples:

```bash
gws schema drive.files.list
gws schema gmail.users.messages.get
gws schema calendar.events.insert
gws schema sheets.spreadsheets.values.get
```

## Patterns to prefer

- Read-only exploration first:
  - `list`
  - `get`
  - `search`-style queries via `--params`
- Mutations only after confirmation:
  - `create`
  - `insert`
  - `update`
  - `patch`
  - `delete`
  - `move`

## Good fits for this skill

Use the `googleworkspace-cli` skill when the user wants:

- direct terminal access to Workspace APIs
- a shell command for a Workspace task
- help choosing the right `gws` service and method
- a safe way to inspect schemas and build payloads

If the user needs a very narrow repetitive workflow, you can still implement it from this generic skill by validating the relevant `gws schema` output first.
