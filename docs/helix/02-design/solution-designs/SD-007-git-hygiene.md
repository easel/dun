---
dun:
  id: SD-007
  depends_on:
  - F-005
---
# Solution Design: Git Hygiene and Hook Checks

## Problem

Agents need fast feedback on working tree cleanliness and hook status before
committing changes.

## Goals

- Detect dirty working trees deterministically.
- Run configured hook tools when available.
- Warn when hook config exists but tooling is missing.
- Keep checks fast and local-only.

## Approach

1. Run `git status --porcelain` to detect dirty paths.
2. Detect hook configuration (lefthook, pre-commit) in a fixed order.
3. If a tool is available, run its non-interactive command.
4. Emit actionable results with next steps.

## Components

- Git Status Checker: detects uncommitted changes.
- Hook Detector: finds configured hook tools.
- Hook Runner: executes hook commands deterministically.
- Reporter: formats pass/warn/fail output.

## Data Flow

1. Git checker identifies dirty paths and emits issues.
2. Hook detector selects the configured tool.
3. Hook runner executes the tool or emits a warning.
4. Reporter emits final results and next actions.

## Open Questions

- Should hook execution be opt-in via config?
- Which additional hook tools should be supported later?
