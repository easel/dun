---
dun:
  id: F-016
  depends_on:
    - helix.prd
  review:
    self_hash: 9cbb749b94bf2884a02ba7fa82990314dca802ce3abb5d7737b3e2be5a2779af
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-016 Doc DAG + Review Stamps

## Summary

Track documentation dependencies via frontmatter-defined DAGs, detect missing
and stale artifacts, and drive updates through prompt envelopes and review
stamps with deterministic ordering.

## Requirements

- Parse `dun` frontmatter in Markdown documents to define:
  - Stable `id`
  - `depends_on` relationships
  - Optional `prompt` and `inputs`
  - Optional `review` stamps (`self_hash`, `deps`)
- Support optional graph files for required roots, ID mappings, and default
  prompt settings.
- Resolve dynamic inputs deterministically with the following selectors:
  - `node:<id>` (explicit node reference)
  - `refs:<id>` (IDs extracted from a referenced document)
  - `code_refs:<id>` (ID references in code paths)
  - `paths:<glob>` (explicit path globs)
- Provide an ID map for `refs` and `node` resolution (e.g. `US-{id}` to
  `docs/helix/01-frame/user-stories/US-{id}-*.md`).
- Compute deterministic content hashes that include frontmatter (excluding
  `dun.review`) plus document body.
- Determine stale documents when parent hashes differ from `review.deps` or
  when `review` is missing.
- Determine missing documents when required roots or required descendants are
  absent.
- Emit prompt envelopes for missing or stale documents using frontmatter
  prompt settings or graph defaults, and include resolved inputs.
- Provide `dun stamp` to update `dun.review` fields in docs.
- Keep output deterministic for a given repo state (ordering, hashing, and
  prompt generation).
- Prompt templates must require a "Gaps & Conflicts" section and instruct the
  agent to flag unresolved conflicts before proceeding.

## Inputs

- Markdown documents with `dun` frontmatter.
- Optional graph files under `.dun/graphs/*.yaml`.

## Acceptance Criteria

- When a parent document changes, all descendants are marked stale until they
  are stamped with updated parent hashes.
- When a required document is missing, Dun reports it as missing with a prompt
  to create it.
- `dun stamp` updates `dun.review.self_hash` and `dun.review.deps` for each
  stamped document.
- Prompt envelopes include parent context inputs by default.
- Prompt envelopes include resolved requirements, ADRs, and code references
  when selectors are configured.
- Results are stable across repeated runs with the same repo state.
- Prompt envelopes require a "Gaps & Conflicts" section before any
  implementation steps.

## Gaps & Conflicts

- Conflicts: none identified in the provided inputs.
- Missing definition of the graph schema (fields for required roots, ID maps,
  and prompt defaults) and precedence between graph defaults and frontmatter.
- Missing rules for handling duplicate IDs, missing IDs, and dependency
  cycles in the DAG.
- Missing definition of how `refs` and `code_refs` are extracted (token
  patterns, scope, and supported file types).
- Missing ordering rules for outputs at the same DAG level (path sort, ID
  sort, or source order).
- Dependencies: output format rules (F-002), doc reconciliation plan ordering
  (F-006), and automation policy (F-007).

## Traceability

- Supports doc reconciliation (F-006) by providing explicit dependency
  tracking and review stamps.
- Supports agent operators by surfacing actionable prompts for stale docs.
