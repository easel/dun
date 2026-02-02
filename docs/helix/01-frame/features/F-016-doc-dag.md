# Feature Spec: F-016 Doc DAG + Review Stamps

## Summary

Track documentation dependencies via frontmatter-defined DAGs, detect missing
and stale artifacts, and drive updates through prompt envelopes and review
stamps.

## Requirements

- Parse `dun` frontmatter in Markdown documents to define:
  - stable `id`
  - `depends_on` relationships
  - `prompt` and `inputs`
  - `review` stamps
- Support an optional graph file for required roots and default prompts.
- Compute deterministic content hashes that include frontmatter (excluding
  `dun.review`).
- Determine stale documents when parent hashes differ from `review.deps`.
- Determine missing documents when required roots or required descendants are
  absent.
- Emit prompt envelopes for missing or stale documents using frontmatter
  prompt settings or graph defaults.
- Provide `dun stamp` to update `dun.review` fields in docs.
- Keep output deterministic for a given repo state.

## Inputs

- Markdown documents with `dun` frontmatter.
- Optional graph files under `.dun/graphs/*.yaml`.

## Acceptance Criteria

- When a parent document changes, all descendants are marked stale until they
  are stamped with updated parent hashes.
- When a required document is missing, Dun reports it as missing with a prompt
  to create it.
- `dun stamp` updates `dun.review.self_hash` and `dun.review.deps`.
- Prompt envelopes include parent context inputs by default.
- Results are stable across repeated runs with the same repo state.

## Traceability

- Supports doc reconciliation (F-006) by providing explicit dependency
  tracking and review stamps.
- Supports agent operators by surfacing actionable prompts for stale docs.
