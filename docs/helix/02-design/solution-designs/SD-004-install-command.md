---
dun:
  id: SD-004
  depends_on:
  - F-004
---
# Solution Design: Install Command

## Problem

Repositories need consistent AGENTS guidance, but manual edits are error-prone
and drift over time.

## Goals

- Insert Dun guidance into AGENTS.md idempotently.
- Support `--dry-run` previews without modifying files.
- Avoid destructive edits by using marker blocks.

## Approach

1. Read AGENTS.md if it exists, otherwise initialize a new file.
2. Detect the Dun marker block and replace or insert it.
3. If `--dry-run` is set, emit the planned change without writing.
4. Otherwise write the updated file to disk.

## Components

- File Reader/Writer: loads and writes AGENTS.md.
- Marker Parser: finds Dun marker blocks.
- Snippet Renderer: formats the guidance block.
- Dry-Run Reporter: emits planned changes.

## Data Flow

1. CLI loads or creates AGENTS.md.
2. Marker parser locates insertion point.
3. Renderer produces the guidance block.
4. Writer or dry-run reporter outputs results.

## Open Questions

- Should custom snippets be supported via config?
- Should insert location be configurable?
