---
dun:
  id: F-007
  depends_on:
    - helix.prd
  review:
    self_hash: bac1592b457dabdacc8cd57755449b797e8d31895bd4a3079fd92f030de2d611
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-007 Automation Slider

## Summary

Expose a single automation mode that governs how Dun executes reconciliation
work (plan-only vs automated completion) while preserving deterministic
results and making the selected mode explicit in outputs.

## Requirements

- Provide a single automation mode selector for `dun check` and reconciliation
  workflows.
- Support automation modes required by the PRD: `plan` and `yolo`.
- `plan` emits reconciliation plans without modifying artifacts.
- `yolo` allows reconciliation to create or update missing artifacts.
- Include the selected mode in prompt envelopes and JSON output so agents
  follow the policy.
- Keep output deterministic for the same repo state and selected mode.

## Inputs

- `docs/helix/01-frame/prd.md` (plan/yolo intent for reconciliation)
- Output format requirements (prompt envelopes, LLM summaries, JSON)
- Reconciliation workflows that apply the automation policy

## Acceptance Criteria

- When automation mode is `plan`, reconciliation output is plan-only with no
  artifact edits.
- When automation mode is `yolo`, reconciliation may create or update missing
  artifacts.
- Prompt envelopes and JSON output include the selected automation mode.
- Output ordering remains deterministic for the same repo state and selected
  mode.

## Gaps & Conflicts

- Conflicts: none identified in the provided inputs.
- Missing definition of how the automation mode is selected (CLI flag, config
  file, or other mechanism) and the precedence rules.
- Missing default mode when no explicit selection is provided.
- Missing behavior for unsupported or unknown automation mode values.
- Missing mapping of automation modes to non-reconciliation checks (for
  example, whether checks can auto-fix).
- Missing guidance on whether LLM summaries must surface the selected
  automation mode alongside prompt envelopes and JSON.
- Missing schema details for how the automation mode is represented in prompt
  envelopes and JSON output.
- Dependencies: reconciliation behavior and output format schemas are defined
  elsewhere.

## Traceability

- Supports PRD goals for doc-to-code reconciliation with plan and yolo modes.
- Supports PRD goals for deterministic, agent-friendly output by including the
  automation mode in prompt envelopes.
