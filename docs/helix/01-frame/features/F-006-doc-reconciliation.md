---
dun:
  id: F-006
  depends_on:
    - helix.prd
  review:
    self_hash: c0cc87165a06f1ff8b90f446f9ae6723319287c7645b0f1659aa3ee618eb4962
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-006 Doc and Code Reconciliation

## Summary

Detect documentation and implementation drift across the Helix stack and emit
a deterministic, ordered reconciliation plan for downstream updates.

## Requirements

- Build a deterministic inventory of artifacts: PRD, feature specs, user
  stories, design docs, ADRs, test plans, and implementation markers.
- Detect drift types: missing artifacts, stale artifacts due to upstream
  changes, and implementation drift relative to docs.
- Emit a structured reconciliation plan as ordered issues with clear next
  steps per artifact.
- Order the plan from upstream to downstream so operators update artifacts in
  dependency order.
- Support automation modes for reconciliation (plan vs yolo) as part of the
  run context.
- Keep the analysis deterministic and reproducible for the same repo state.

## Inputs

- `docs/helix/01-frame/prd.md`
- `docs/helix/01-frame/features/*.md`
- `docs/helix/01-frame/user-stories/*.md`
- `docs/helix/02-design/**/*.md`
- `docs/helix/03-test/test-plan.md`
- Source code paths (e.g., `cmd/`, `internal/`) as needed by the agent
- Automation mode configuration (CLI/config) for plan vs yolo behavior

## Acceptance Criteria

- When a PRD change is detected, Dun emits a drift plan listing impacted
  artifacts in deterministic dependency order.
- The plan includes updates for feature specs, design docs, ADRs, test plans,
  and implementation artifacts; user stories are included when present.
- Drift output is structured as issues with clear next steps.
- Automation mode `plan` emits the reconciliation plan without edits; `yolo`
  allows agents to complete missing artifacts per policy.

## Gaps & Conflicts

- Conflicts: none identified in the provided inputs.
- Missing definition of the drift detection method (hashing, stamps, or diff
  strategy) and how implementation drift is identified.
- Missing ordering rules for artifacts at the same layer and how user stories
  interleave with feature specs.
- Missing plan schema (fields, IDs, severity) and mapping to output formats
  (prompt envelopes vs JSON).
- Missing rules for code scope selection and how much code context to include.
- Dependencies: automation mode policy (F-007), doc dependency tracking
  (F-016), and output format rules (F-002).

## Traceability

- Supports PRD goals for deterministic, agent-friendly output and doc-to-code
  reconciliation with plan and yolo modes.
- Supports US-005 by emitting ordered downstream updates from PRD changes.
