---
dun:
  id: TD-011
  depends_on:
  - US-011
---
# Technical Design: TD-011 Agent Quorum

## Story Reference

**User Story**: US-011 Agent Quorum
**Parent Feature**: F-018 Agent Quorum
**Solution Design**: SD-011 Agent Quorum

## Goals

- Allow multiple harnesses to review each task.
- Require agreement based on quorum strategy before applying changes.
- Provide conflict reporting and escalation options.

## Non-Goals

- Human-in-the-loop UI (log-based escalation is sufficient).

## Technical Approach

### Implementation Strategy

- Run the same prompt through multiple harnesses.
- Normalize responses into a comparable form.
- Apply a quorum strategy: any, majority, unanimous, or N.

### Key Decisions

- Default to sequential execution to control cost; allow parallel mode.
- Keep similarity threshold configurable for conflict detection.

## Component Changes

### Components to Modify

- `internal/dun/quorum.go`: compute agreement and select a response.
- `internal/dun/conflict_detection.go`: compare responses semantically.
- `cmd/dun/main.go`: parse `--quorum`, `--harnesses`, and conflict flags.

### New Components

- Quorum result reporter with conflict details.

## Interfaces and Config

- CLI: `dun loop --quorum`, `--harnesses`, `--cost-mode`, `--escalate`.
- Config: default quorum strategy and similarity threshold.

## Data and State

- Per-iteration quorum summary stored in memory; optional log output.

## Testing Approach

- Unit tests for quorum strategies and similarity thresholds.
- Integration tests for conflict and escalation paths.

## Risks and Mitigations

- **Risk**: Conflicts slow down loops. **Mitigation**: allow `--prefer` and
  `--cost-mode` options.

## Rollout / Compatibility

- Backwards compatible; quorum is opt-in.
