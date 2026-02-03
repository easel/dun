---
dun:
  id: F-001
  depends_on:
    - helix.prd
  review:
    self_hash: 1490cef1b15ca48c530d4f0c4eae1e5e9912c5d3a7dfa1aa0343e7c59290dd78
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-001 Auto-Discovery

## Summary

Detect applicable checks based on repo signals (files and conventions) without
manual configuration.

## Requirements

- Discover core checks from repo signals (files and conventions).
- Detect Go repositories via `go.mod` to enable baseline Go quality checks.
- Detect Helix workflow via `docs/helix/` to enable doc/gate validation checks.
- Produce a deterministic set of checks for the same repo state.
- Require no manual configuration for core discovery.

## Inputs

- Repository root file tree.
- `go.mod` (Go detection signal).
- `docs/helix/` (Helix workflow detection signal).

## Acceptance Criteria

- `dun check` selects Go quality checks when `go.mod` is present.
- `dun check` selects Helix doc/gate checks when `docs/helix/` is present.
- Check IDs and ordering are stable across runs for the same repo state.
- No user configuration is required for core discovery.

## Gaps & Conflicts

- Missing canonical registry of check IDs and their discovery rules (depends on
  F-003 Plugin System).
- Missing ordering rules when multiple signals match (priority and grouping).
- Missing definition of baseline Go quality checks to enable (depends on
  F-014 Go Quality Checks).
- Missing definition of how discovery hooks into doc drift detection (depends
  on F-006 Doc Reconciliation and F-016 Doc DAG).
- No conflicts identified in the provided inputs.

## Traceability

- Supports PRD goals for "one command that discovers and runs the right
  checks" and deterministic output.
- Supports PRD scope for built-in Helix doc validation and baseline Go quality
  checks by using repo signals for discovery.
