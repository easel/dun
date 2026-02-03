---
dun:
  id: TD-004
  depends_on:
  - US-004
---
# Technical Design: TD-004 Install Command

## Story Reference

**User Story**: US-004 Install Command
**Parent Feature**: F-004 Install Command
**Solution Design**: SD-004 Install Command

## Goals

- Provide `dun install` that adds agent guidance to AGENTS.md.
- Keep edits idempotent and non-destructive.
- Support `--dry-run` for previewing changes.

## Non-Goals

- Installing binaries or system dependencies (covered by installer/updater).

## Technical Approach

### Implementation Strategy

- Use a marker block in AGENTS.md to delimit Dun-managed content.
- If AGENTS.md does not exist, create it with a minimal header plus block.
- If the block exists, replace it in place to keep edits idempotent.

### Key Decisions

- Use `<!-- DUN:BEGIN -->` / `<!-- DUN:END -->` markers for safe updates.
- Keep the installed snippet short and self-contained.

## Component Changes

### Components to Modify

- `internal/dun/install.go`: read/update AGENTS.md using marker blocks.
- `cmd/dun/main.go`: add `dun install` command flags.

### New Components

- None.

## Interfaces and Config

- CLI: `dun install` and `dun install --dry-run`.

## Data and State

- AGENTS.md is the only persistent artifact.

## Testing Approach

- File-based tests for create/replace/idempotent updates.
- Dry-run tests to ensure no writes occur.

## Risks and Mitigations

- **Risk**: Overwriting user content. **Mitigation**: marker-based edits only.

## Rollout / Compatibility

- Safe for existing repos; no changes unless install is invoked.
