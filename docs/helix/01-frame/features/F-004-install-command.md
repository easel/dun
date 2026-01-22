# Feature Spec: F-004 Install Command

## Summary

Provide a `dun install` command that inserts AGENTS.md guidance for agent loops.

## Requirements

- Idempotently insert a Dun tool snippet in AGENTS.md.
- Support `--dry-run` to show planned changes.
- Avoid destructive edits; use marker blocks.

## Acceptance Criteria

- `dun install` writes AGENTS.md if missing.
- `dun install --dry-run` shows planned steps without changes.
- Re-running `dun install` is safe and idempotent.
