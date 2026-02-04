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

- Provide `dun check --prompt` for external agents (task list only).
- Provide `dun task <task-id> --prompt` to fetch the full task prompt.
- Provide `dun loop` for in-process iteration with a harness.
- Ensure each iteration runs with fresh context and clear prompts.

## Non-Goals

- Multi-agent consensus (covered by agent quorum).

## Technical Approach

### Implementation Strategy

- Run checks to build a task list and render a decision prompt.
- Select a task ID, then load its full prompt via `dun task`.
- `dun task` re-runs checks and looks up the task in the current result set;
  task IDs are valid only for the current repo state.
- Invoke the selected harness with the prompt and capture response.
- Apply responses via `dun respond`, then re-run checks.
- Stop when all checks pass or max iterations reached.

### Key Decisions

- Use a fixed prompt envelope to keep harness inputs stable.
- Treat each loop iteration as a clean run with no hidden state.
- Task IDs are stable within a single repo state:
  `<check-id>@<state>` or `<check-id>#<n>@<state>` for issue-indexed tasks
  (1-based). `state` is a short hash of git HEAD + working tree status.
- Decision prompt limits: top 10 tasks per check; summary <= 200 bytes;
  reason <= 160 bytes; truncation uses `...`.

## Component Changes

### Components to Modify

- `internal/dun/engine.go`: expose prompt generation for the loop.
- `internal/dun/agent.go`: run harness with timeout and capture output.
- `internal/dun/respond.go`: apply agent changes deterministically.
- `cmd/dun/main.go`: add `check --prompt` and `loop` flags.

### New Components

- Loop orchestration helper for iteration state and exit criteria.

## Interfaces and Config

- CLI: `dun check --prompt`, `dun task <task-id> --prompt`,
  `dun loop --max-iterations`, `--automation`.
- Config: harness command and timeout defaults.

## Data and State

- Optional temp files for harness I/O; no persistent state.

## Testing Approach

- Unit tests for prompt generation and response parsing.
- Integration tests for a dry-run loop (no external harness).
- Tests for task ID parsing, stale/invalid task IDs, and prompt truncation.

## Risks and Mitigations

- **Risk**: Infinite loops. **Mitigation**: max-iterations guard and clear
  exit signals.

## Rollout / Compatibility

- Backwards compatible; loop is an additive capability.
