# Solution Design: Doc DAG + Review Stamps

## Problem

Documentation dependencies are implicit, so Dun cannot reliably identify
which artifacts are missing or stale when upstream docs change.

## Goals

- Encode dependencies in doc frontmatter.
- Detect missing and stale docs deterministically.
- Drive updates through prompt envelopes.
- Persist review stamps in documents (no external cache).

## Approach

1. **Frontmatter parsing**: read `dun` blocks from Markdown to register nodes.
2. **Optional graph defaults**: load `.dun/graphs/*.yaml` for required roots
   ID mappings, and default prompts for missing docs.
3. **Input resolution**: resolve deterministic inputs via selectors:
   `node:<id>`, `refs:<id>`, `code_refs:<id>`, `paths:<glob>`.
4. **Hashing**: compute a stable hash of each doc including frontmatter,
   excluding `dun.review`.
5. **Staleness**: compare parent hashes to `dun.review.deps`.
6. **Missing detection**: flag required roots or required descendants with no
   files.
7. **Prompting**: emit prompts for missing or stale docs with parent inputs
   and require gaps/conflicts to be flagged before implementation steps.
8. **Stamping**: `dun stamp` writes updated review hashes to frontmatter.

## Components

- **Frontmatter Reader**: extracts `dun` config and review stamps.
- **Doc Graph Builder**: builds the DAG from frontmatter + graph defaults.
- **Input Resolver**: expands selectors into ordered input paths.
- **Hasher**: computes doc content hashes.
- **Doc-DAG Check**: emits missing/stale issues and prompts.
- **Stamp Command**: updates review stamps in files.

## Data Flow

1. `dun check` runs `doc-dag` check.
2. Frontmatter reader registers nodes and dependencies.
3. Graph builder adds required roots/defaults and ID map.
4. Input resolver expands selectors to a sorted input list.
5. Hasher computes current hashes.
6. Staleness/missing detection runs.
7. Prompt envelopes are emitted for actionable nodes.

## Data Model

- **Node**: id, path, depends_on, prompt, inputs, review
- **Input selector**: node, refs, code_refs, paths
- **Review**: self_hash, deps map, reviewed_at
- **Edge**: parent -> child from `depends_on`

## Hashing Rules

- Hash includes:
  - Markdown body
  - Frontmatter contents excluding `dun.review`
- Normalize line endings to `\n`.
- Use a stable YAML encoding for the remaining frontmatter.

## Interface Changes

- New check type: `doc-dag`.
- New command: `dun stamp`.
- Optional graph files: `.dun/graphs/*.yaml`.
- Optional ID map in graph file for resolving refs.

## Files (Planned)

- `internal/dun/doc_dag.go` (graph build + staleness detection)
- `internal/dun/frontmatter.go` (parse/serialize frontmatter)
- `internal/dun/input_resolver.go` (resolve selectors to inputs)
- `internal/dun/hash.go` (doc hashing)
- `internal/dun/stamp.go` (stamp logic)
- `cmd/dun/main.go` (wire `dun stamp`)

## Open Questions

- Should `reviewed_at` be used for display only, or validated?
- How should collection nodes (e.g., US-* chains) map IDs deterministically?
