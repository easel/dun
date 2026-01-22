# ADR-006: Automation Slider for Reconciliation

## Status

Proposed

## Context

Dun should support both human-in-the-loop and autonomous execution when
reconciling documentation and code. A single hardcoded mode is insufficient.

## Decision

Introduce an automation mode flag with four levels: manual, plan, auto, yolo.
The selected mode is injected into prompt envelopes and used to guide agent
behavior. Manual and plan modes do not make edits without explicit approval.

## Consequences

- Prompts must include the automation mode.
- Agents can follow a deterministic policy per run.
- Yolo mode increases risk of overfitting but accelerates convergence.
