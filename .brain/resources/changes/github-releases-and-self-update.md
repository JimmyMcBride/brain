---
created: "2026-04-11T00:00:00Z"
source: legacy_capture
title: GitHub Releases And Self-Update
type: change
updated: "2026-04-11T05:25:43Z"
---
# GitHub Releases And Self-Update

Added version and update commands, GitHub release packaging, checksum-verified downloads, platform-specific asset selection, in-place replacement when writable, and fallback installation to `~/.local/bin/brain`.

The project-local redesign was also installed directly from source into `~/.local/bin/brain` so it takes precedence over the older `/usr/local/bin/brain` copy on PATH. That is the expected local upgrade path before a tagged release exists.
