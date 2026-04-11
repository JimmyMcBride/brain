# gws Quickstart

Primary repo: `https://github.com/googleworkspace/cli`

The CLI binary is `gws`.

## Install

Preferred install paths:

- GitHub Releases: `https://github.com/googleworkspace/cli/releases`
- npm:
  ```bash
  npm install -g @googleworkspace/cli
  ```
- Homebrew:
  ```bash
  brew install googleworkspace-cli
  ```
- Build from source:
  ```bash
  cargo install --git https://github.com/googleworkspace/cli --locked
  ```

## First-run auth

Local interactive setup:

```bash
gws auth setup
gws auth login
```

Useful auth alternatives:

- `GOOGLE_WORKSPACE_CLI_TOKEN`
- `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE`

For manual OAuth setup, the upstream README documents using a desktop OAuth client and saving the client JSON under `~/.config/gws/client_secret.json`.

## Core commands

Inspect the service:

```bash
gws drive --help
gws gmail --help
gws calendar --help
gws sheets --help
```

Inspect the exact method schema:

```bash
gws schema drive.files.list
gws schema calendar.events.insert
gws schema sheets.spreadsheets.values.get
```

Run commands:

```bash
gws drive files list --params '{"pageSize": 10}'
gws sheets spreadsheets create --json '{"properties":{"title":"Q1 Budget"}}'
gws drive files list --params '{"pageSize": 100}' --page-all
```

## Important flags

- `--params '{"key":"value"}'`
- `--json '{"key":"value"}'`
- `--output /path/to/file`
- `--page-all`
- `--format json|table|yaml|csv`
- `--dry-run`

## Notes

- `gws` builds its command tree from Google's Discovery Service at runtime.
- Prefer `gws schema ...` before constructing request payloads.
- Keep JSON output when an agent or script will post-process the result.
