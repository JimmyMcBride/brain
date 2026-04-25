---
updated: "2026-04-25T05:21:15Z"
---
# Implement Context Effectiveness Reporting Command

Added `brain context effectiveness` as a higher-level report over recorded packet telemetry. The command summarizes packet use, cache behavior, budget pressure, outcome links, likely misses from repeated omitted docs, known telemetry gaps, and recommended packet-shaping follow-ups.

## Verification

- `go test ./...`
- `go build ./...`
- `go test -race ./...`
