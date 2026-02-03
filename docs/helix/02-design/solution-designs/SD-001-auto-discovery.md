---
dun:
  id: SD-001
  depends_on:
    - F-001
  review:
    self_hash: 60747e704bc4549a4d605388f00bea3546aefb55da92b7b87ca8a7f05ff3f3a5
    deps:
      F-001: 1490cef1b15ca48c530d4f0c4eae1e5e9912c5d3a7dfa1aa0343e7c59290dd78
---
# Solution Design: Auto-Discovery

## Problem

Manual configuration slows agent loops and leads to inconsistent check
selection across repos.

## Goals

- Detect repo signals (`go.mod`, `docs/helix/`) deterministically.
- Activate the correct plugins without user configuration.
- Keep discovery fast and local.

## Inputs

- Repository root file tree.
- `go.mod` (Go detection signal).
- `docs/helix/` (Helix workflow detection signal).

## Gaps & Conflicts

- Missing canonical registry of check IDs and their discovery rules (depends on
  F-003 Plugin System).
- Missing ordering rules when multiple signals match (priority and grouping).
- Missing definition of baseline Go quality checks to enable (depends on
  F-014 Go Quality Checks).
- Missing definition of how discovery hooks into doc drift detection (depends
  on F-006 Doc Reconciliation and F-016 Doc DAG).
- No conflicts identified in the provided inputs.

## Approach

1. Scan the repo root for known signal files and paths in a fixed order.
2. Activate plugins whose signals match the repo state.
3. Merge checks into a single plan and sort by a stable key.

## Components

- Signal Scanner: checks for known files and paths.
- Plugin Registry: stores embedded manifests.
- Check Planner: builds the deterministic check list.
- Sorter: enforces stable ordering.

## Data Flow

1. Scanner evaluates repo signals.
2. Registry activates matching plugins.
3. Planner merges checks into an ordered plan.
4. Reporter emits the planned checks.
