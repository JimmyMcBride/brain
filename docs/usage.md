# Usage

## Initialization

```bash
brain init
brain doctor
```

## Note creation

```bash
brain add "Client migration" --section Projects --type project
brain add "TLS notes" --section Resources --type resource --template resource.md
brain capture "Follow up with vendor" --body "Need pricing and rollout dates."
brain daily
```

## Reading and editing

```bash
brain read Projects/client-migration.md
brain edit Projects/client-migration.md --set status=active
brain edit Projects/client-migration.md --editor nvim
brain move Projects/client-migration.md Archives/
```

## Retrieval

```bash
brain reindex
brain find migration
brain find --type project
brain search "vendor rollout timeline"
```

## Safety

```bash
brain history
brain undo
brain organize
brain organize --apply
```

## Content workflow

```bash
brain content seed Projects/client-migration.md
brain content gather Projects/client-migration.md -n 5
brain content outline Projects/client-migration.md -n 5
brain content publish Projects/client-migration.md --channel blog --repurpose thread
```

## Skills

```bash
brain skills targets --scope both --agent codex --agent claude --project .
brain skills install --scope global --agent codex
brain skills install --scope local --agent codex --project .
brain skills install --scope both --agent codex --agent zed --project .
brain skills install --skill-root /path/to/custom/skills --mode copy
```
