# Solution Design: Auto-Discovery

## Problem

Manual configuration slows agent loops and leads to inconsistent check
selection across repos.

## Goals

- Detect repo signals (`go.mod`, `docs/helix/`) deterministically.
- Activate the correct plugins without user configuration.
- Keep discovery fast and local.

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

## Open Questions

- How should mixed-language monorepos influence plugin activation?
- Should discovery cache results between runs?
