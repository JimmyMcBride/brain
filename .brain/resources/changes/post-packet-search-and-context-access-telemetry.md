---
status: implemented
title: Post-Packet Search And Context Access Telemetry
type: change
updated: "2026-04-25T06:05:15Z"
---
# Post-Packet Search And Context Access Telemetry

## Summary

Brain records two new context-effectiveness signals after a compiled packet exists in an active session:

- `post_packet_search` for `brain search`, `brain search --explain`, and `brain search --inject`
- `context_access_recorded` for low-risk read/search-like commands routed through `brain session run`

## Details

Search telemetry stores compact metadata only: query, limit, result count, explain/inject mode flags, and top result paths/headings without snippets.

Context-access telemetry classifies direct `cat`, `sed`, `head`, `tail`, `nl`, `less`, `more`, `rg`, `grep`, and `git grep` invocations, stores the command family and command string, and records only existing repo-local file or directory paths.

`brain context effectiveness` now reports post-packet search counts, Brain-routed context-access counts, and likely-miss evidence for omitted markdown docs later accessed during the same packet window. `brain context explain --last` renders those downstream outcomes for packet-level inspection.
