# Usage

`docs/usage.md` is the operator manual. It explains how to use `brain` day to day.

Read [`../README.md`](../README.md) first if you want the quick product overview. Read [`architecture.md`](architecture.md) if you want internals.

## Common workflows

### 1. Set up a vault

```bash
brain init
brain doctor
```

Use this first on a new machine or when pointing `brain` at a new vault.

### 2. Create notes quickly

```bash
brain add "Client migration" --section Projects --type project
brain add "TLS notes" --section Resources --type resource --template resource.md
brain capture "Follow up with vendor" --body "Need pricing and rollout dates."
brain daily
```

Use `add` for durable notes with a template, `capture` for fast intake, and `daily` for a dated daily note.

### 3. Read, edit, and move notes

```bash
brain read Projects/client-migration.md
brain edit Projects/client-migration.md --set status=active
brain edit Projects/client-migration.md --editor nvim
brain move Projects/client-migration.md Archives/
```

### 4. Build and query the index

```bash
brain reindex
brain find migration
brain find --type project
brain search "vendor rollout timeline"
```

Use `find` for direct vault/path/metadata matching. Use `search` when you want hybrid retrieval over indexed chunks.

### 5. Use the safety system

```bash
brain history
brain undo
brain organize
brain organize --apply
```

Use `history` and `undo` whenever you want a reversible workflow around note changes.

### 6. Turn notes into content

```bash
brain content seed Projects/client-migration.md
brain content gather Projects/client-migration.md -n 5
brain content outline Projects/client-migration.md -n 5
brain content publish Projects/client-migration.md --channel blog --repurpose thread
```

### 7. Install agent skills

```bash
brain skills targets --scope both --agent codex --agent claude --project .
brain skills install --scope global --agent codex
brain skills install --scope local --agent codex --project .
brain skills install --scope both --agent codex --agent zed --project .
brain skills install --skill-root /path/to/custom/skills --mode copy
```

Use this when you want a global or project-local `brain` skill available to coding agents.

### 8. Add project context to a repo

```bash
brain context install --project . --agent codex --agent openclaw
brain context refresh --project .
brain context refresh --project . --dry-run
```

This generates a root `AGENTS.md`, `.brain/context/*`, `.brain/policy.yaml`, and optional agent wrappers.

### 9. Enforce a session in a repo

```bash
brain session start --project . --task "tighten search ranking"
brain session validate --project .
brain session run --project . -- go test ./...
brain session run --project . -- go build ./...
brain session finish --project . --summary "search update complete"
brain session abort --project . --reason "switching tasks"
```

Use sessions when you want the repo to require durable note updates, reindexing, and recorded verification commands.

## Command intent

- `init`, `doctor`: create and validate the local environment
- `add`, `capture`, `daily`: create new notes
- `read`, `edit`, `move`: work on existing notes
- `reindex`, `find`, `search`: retrieve knowledge
- `history`, `undo`, `organize`: operate safely
- `content *`: turn notes into publishing assets
- `skills *`: install the `brain` skill for agents
- `context *`: install project-local agent context
- `session *`: enforce project workflow rules
