---
dun:
  id: TD-009
  depends_on:
  - US-009
---
# Technical Design: TD-009 Autonomous Iteration

## Story Reference

**User Story**: US-009 Run Autonomous Iteration Loop
**Parent Feature**: F-017 Autonomous Iteration
**Solution Design**: (not yet documented)

## Goals

- Provide `dun check --prompt` for external agents.
- Provide `dun loop` for in-process iteration with a harness.
- Ensure each iteration runs with fresh context and clear prompts.

## Non-Goals

- Multi-agent consensus (covered by agent quorum).

## Technical Approach

### Implementation Strategy

- Run checks to build a work list and render a prompt.
- Invoke the selected harness with the prompt and capture response.
- Apply responses via `dun respond`, then re-run checks.
- Stop when all checks pass or max iterations reached.

### Key Decisions

- Use a fixed prompt envelope to keep harness inputs stable.
- Treat each loop iteration as a clean run with no hidden state.

## Component Changes

### Components to Modify

- `internal/dun/engine.go`: expose prompt generation for the loop.
- `internal/dun/agent.go`: run harness with timeout and capture output.
- `internal/dun/respond.go`: apply agent changes deterministically.
- `cmd/dun/main.go`: add `check --prompt` and `loop` flags.

### New Components

- Loop orchestration helper for iteration state and exit criteria.

## Interfaces and Config

- CLI: `dun check --prompt`, `dun loop --max-iterations`, `--automation`.
- Config: harness command and timeout defaults.

## Data and State

- Optional temp files for harness I/O; no persistent state.

## Testing Approach

- Unit tests for prompt generation and response parsing.
- Integration tests for a dry-run loop (no external harness).

## Risks and Mitigations

- **Risk**: Infinite loops. **Mitigation**: max-iterations guard and clear
  exit signals.

## Rollout / Compatibility

- Backwards compatible; loop is an additive capability.
