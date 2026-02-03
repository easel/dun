---
dun:
  id: TD-013
  depends_on:
  - US-013
  - ADR-007
---
# Technical Design: TD-013 Doc DAG and Review Stamps

## Story Reference

**User Story**: US-013 Doc DAG and Review Stamps
**Parent Feature**: F-016 Doc DAG + Review Stamps
**Solution Design**: SD-013 Doc DAG

## Goals

- Build a DAG of Helix docs based on frontmatter dependencies.
- Detect missing and stale documents.
- Provide a `dun stamp` command to update review stamps.

## Non-Goals

- Automatic document generation (agents handle content).

## Technical Approach

### Implementation Strategy

- Parse `dun` frontmatter from docs and build a graph keyed by doc ID.
- Compute canonicalized hashes (per ADR-007) and compare them to
  `dun.review.deps` to identify stale nodes.
- Treat docs without review stamps as stale.
- Provide `dun stamp` to write `dun.review.self_hash` and `dun.review.deps`
  (optionally `reviewed_at`).

### Key Decisions

- Use YAML frontmatter so relationships remain in the docs themselves.
- Canonicalize frontmatter before hashing for determinism (ADR-007).
- Restrict scope to `docs/helix/**` for predictable inputs.

## Component Changes

### Components to Modify

- `internal/dun/frontmatter.go`: parse and serialize frontmatter.
- `internal/dun/doc_dag.go`: build the DAG and compute stale/missing docs.
- `internal/dun/stamp.go`: apply stamp updates.
- `cmd/dun/main.go`: add `stamp` command.

### New Components

- Test fixtures for missing and cascade scenarios.

## Interfaces and Config

- CLI: `dun stamp [--all] [paths...]`.
- Config: optional graph root and exclude patterns.

## Data and State

- Frontmatter includes `dun.review.self_hash`, `dun.review.deps`, and optional
  `reviewed_at`.

## Testing Approach

- Unit tests for frontmatter parsing/serialization.
- Graph tests for missing and stale detection.
- CLI tests for `dun stamp` behavior.

## Risks and Mitigations

- **Risk**: Overwriting manual frontmatter edits. **Mitigation**: update only
  stamp fields and preserve other metadata.

## Rollout / Compatibility

- Backwards compatible; docs without frontmatter are treated as missing and
  unstamped docs are treated as stale until updated.
