---
dun:
  id: ADR-006
  depends_on:
    - helix.prd
    - helix.architecture
  review:
    self_hash: 580b9bce139ce48fb7138f3bfab909b1019ee803b04c5a11a2b591952026603b
    deps:
      helix.architecture: a090dcf41ac7011e8f723d9f7e6a4cc992e618e713d95374791cecec4436c309
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# ADR-006: Automation Slider for Reconciliation

## Status

Proposed

## Context

Dun is a local CLI that emits deterministic prompt envelopes and can optionally
run agents. Reconciliation spans documentation and code, so operators need a
clear, repeatable policy for how much automation is allowed. A single, fixed
mode would not serve both cautious review loops and faster iterations.

## Decision

Define an automation mode with four levels: manual, plan, auto, yolo. The
selected mode is part of the run context, included in prompt envelopes, and
enforced by the automation policy that gates agent behavior. Manual and plan
do not make edits without explicit approval.

## Gaps & Conflicts

- Conflicts: none identified in the provided inputs.
- Missing inputs: default mode, configuration source (CLI flag vs config),
  and the precise behavioral difference between auto and yolo.
- Dependencies: prompt renderer/emitter must carry the mode; automation policy
  must enforce the gate; optional agent runner must honor the policy.

## Consequences

- Prompts must include the automation mode.
- Agents can follow a deterministic policy per run.
- Autonomous mode increases risk of overfitting but accelerates convergence.

## Verification

- TBD. No specific test or validation criteria are defined in the inputs.
