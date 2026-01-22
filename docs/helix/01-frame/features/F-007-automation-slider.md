# Feature Spec: F-007 Automation Slider

## Summary

Allow Dun to operate with varying autonomy, from manual approval to yolo
execution, when reconciling documentation and code.

## Requirements

- Provide a CLI flag to set automation mode.
- Default automation mode is `auto`, with overrides via `dun.yaml`.
- Modes:
  - `manual`: prompt-only, human approval for each change.
  - `plan`: emit a detailed plan without modifying artifacts.
  - `auto`: agent executes changes but asks when blocked.
  - `yolo`: agent may create/modify artifacts to declare completeness.
- Automation mode must be included in prompts so agents follow the policy.

## Acceptance Criteria

- `dun check --automation=plan` emits reconciliation plans only.
- `dun check --automation=yolo` allows the agent to fill missing artifacts.
- Prompt envelopes include the selected automation mode.
