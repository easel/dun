# Feature Spec: F-001 Auto-Discovery

## Summary

Detect applicable checks based on repo signals (files and conventions) without
manual configuration.

## Requirements

- Detect Go repositories via `go.mod`.
- Detect Helix workflow via `docs/helix/`.
- Produce a deterministic set of checks for the repo state.

## Acceptance Criteria

- `dun check` lists checks based on detected repo signals.
- Check IDs and ordering are stable across runs for the same repo state.
- No user configuration is required for core discovery.
