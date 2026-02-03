---
dun:
  id: TD-006
  depends_on:
  - US-006
  - ADR-006
---
# Technical Design: TD-006 Automation Slider

## Story Reference

**User Story**: US-006 Automation Slider
**Parent Feature**: F-007 Automation Slider
**Solution Design**: (not yet documented) See ADR-006
**ADR**: ADR-006 Automation Slider

## Goals

- Provide automation modes: manual, plan, auto, yolo.
- Allow users to scale autonomy without changing commands.
- Ensure risky operations require explicit opt-in.

## Non-Goals

- Full policy engine or RBAC.

## Technical Approach

### Implementation Strategy

- Define an `AutomationMode` enum and store it in config/CLI flags.
- Gate actions (agent edits, command execution, file writes) based on the mode.
- Emit clear messaging when actions are blocked by policy.

### Key Decisions

- Default mode is `auto` to balance safety and automation.
- `yolo` explicitly disables safety prompts and runs with maximum autonomy.

## Component Changes

### Components to Modify

- `internal/dun/rules.go`: define policy gates for actions.
- `internal/dun/engine.go`: enforce policy before executing checks or fixes.
- `cmd/dun/main.go`: add `--automation` flag for relevant commands.

### New Components

- None.

## Interfaces and Config

- CLI: `--automation manual|plan|auto|yolo`.
- Config: `.dun/config.yaml` default automation mode.

## Data and State

- Mode is runtime-only; no persisted state beyond config.

## Testing Approach

- Unit tests for each automation mode gate.
- Integration tests ensuring blocked actions produce actionable warnings.

## Risks and Mitigations

- **Risk**: Users bypass safety unintentionally. **Mitigation**: require
  explicit `--automation yolo` for destructive actions.

## Rollout / Compatibility

- Backwards compatible; default behavior matches current mode.
