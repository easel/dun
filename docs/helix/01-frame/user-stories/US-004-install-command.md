# US-004: Install AGENTS Guidance

As a maintainer, I want a `dun install` command to seed AGENTS guidance so
agent loops follow the correct workflow without manual setup.

## Acceptance Criteria

- `dun install` writes AGENTS.md if it is missing.
- `dun install --dry-run` shows planned changes without writing files.
- Re-running `dun install` is safe and idempotent.
- The insertion uses marker blocks and avoids destructive edits.
