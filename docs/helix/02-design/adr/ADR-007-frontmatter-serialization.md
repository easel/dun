---
dun:
  id: ADR-007
  depends_on:
  - helix.prd
  - helix.architecture
---
# ADR-007: Frontmatter YAML Canonicalization for Doc Hashing

## Status

Proposed

## Context

Doc hashing includes frontmatter. YAML serialization is not deterministic by
default: key order, indentation, and line wrapping can change without any
semantic edit. We need a canonical form so hashes only change when the
frontmatter meaning changes.

## Decision

Canonicalize frontmatter before hashing:

- Parse YAML into a structured form and re-encode it for hashing.
- Remove the `dun.review` subtree before hashing.
- Sort mapping keys lexicographically at every level.
- Preserve sequence order (arrays are not sorted).
- Emit with stable formatting: 2-space indentation, block style where
  available, and no line wrapping.
- Normalize line endings to `\n` and ensure a trailing newline.
- Treat missing frontmatter as empty.
- Invalid YAML or unsupported encodings are errors and block hashing.

## Consequences

- Hashes change only when the semantic frontmatter (excluding `dun.review`)
  changes.
- Comments and key ordering do not affect hashes.
- All hash computations must use the canonicalization helper for consistency.

## Verification

- Unit test `TestHashCanonicalizesFrontmatter` in
  `internal/dun/hash_test.go` validates that equivalent YAML yields identical
  hashes.
