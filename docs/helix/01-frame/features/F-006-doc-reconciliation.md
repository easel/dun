# Feature Spec: F-006 Doc and Code Reconciliation

## Summary

Detect documentation drift across the Helix stack and propose downstream
changes from PRD to implementation.

## Requirements

- Compare PRD, feature specs, user stories, design docs, ADRs, test plans, and
  implementation for drift.
- Emit a structured plan that lists required downstream updates.
- Support both documentation-only drift and implementation drift.
- Keep the analysis deterministic and reproducible.

## Inputs

- `docs/helix/01-frame/prd.md`
- `docs/helix/01-frame/features/*.md`
- `docs/helix/01-frame/user-stories/*.md`
- `docs/helix/02-design/**/*.md`
- `docs/helix/03-test/test-plan.md`
- Source code paths (e.g., `cmd/`, `internal/`) as needed by the agent

## Acceptance Criteria

- When a PRD change is detected, Dun emits a drift plan listing impacted
  artifacts in order.
- Drift output is structured as issues with clear next steps.
- The plan traces updates all the way to implementation and tests.
