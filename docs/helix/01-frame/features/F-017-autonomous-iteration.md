---
dun:
  id: F-017
  depends_on:
    - helix.prd
  review:
    self_hash: 7bee4b95105a9318d51a42ae77fa37aedbc5f28d82f3cb2a155a97eedf809f8f
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-017 Autonomous Iteration Loop

## Summary

Enable a deterministic autonomous iteration loop that emits a structured work
list (`dun check --prompt`) and runs an embedded loop (`dun loop`) that calls a
configured agent harness until exit conditions are met.

## Requirements

- Provide `dun check --prompt` to emit a deterministic, structured work list
  prompt for an external agent.
- Provide `dun loop` to run an embedded iteration that:
  - Runs checks, renders a prompt, calls the configured harness, applies
    responses, and re-runs checks.
- Support harness selection for `codex`, `claude`, and `gemini` via flags or
  config.
- Support automation modes (`manual`, `plan`, `auto`, `yolo`) and include the
  selected mode in prompts sent to the harness.
- Ensure each iteration runs with fresh context (no hidden state carried
  between iterations).
- Exit when all checks pass, `--max-iterations` is reached, a user aborts, or
  an explicit exit signal is received.
- Provide `--dry-run` to print the prompt without invoking the harness.
- Provide `--verbose` to log prompts and responses for auditability.
- Ensure `dun install` adds AGENTS guidance for the check/loop pattern.
- Ensure `dun help` documents `check --prompt`, `loop`, and key options.
- Keep output deterministic for the same repo state and configuration.

## Inputs

- `docs/helix/01-frame/prd.md` (product intent and automation policy intent).
- `docs/helix/01-frame/user-stories/US-009-autonomous-iteration.md`.
- `docs/helix/02-design/technical-designs/TD-009-autonomous-iteration.md`.
- `docs/design/contracts/API-001-dun-cli.md` (CLI options and exit codes).
- `.dun/config.yaml` (harness and automation defaults).
- `AGENTS.md` (agent workflow guidance).

## Loop Flow

1. Run checks and build the work list.
2. If all checks pass, exit successfully.
3. Render the prompt and include the selected automation mode.
4. If `--dry-run`, emit the prompt and exit without calling a harness.
5. Call the configured harness, capture its response, and apply changes.
6. Repeat until an exit condition is met.

## Acceptance Criteria

- `dun check --prompt` outputs a deterministic work list prompt for an external
  agent.
- `dun loop` runs an embedded loop calling a configurable agent harness.
- The loop supports multiple harnesses: `claude`, `gemini`, `codex`.
- Each iteration spawns fresh context to prevent drift.
- The loop exits when all checks pass or max iterations is reached.
- Yolo mode passes appropriate flags to the harness for autonomous operation.
- `dun install` adds agent documentation to AGENTS.md explaining the pattern.
- `dun help` documents the check --prompt and loop commands.

## Gaps & Conflicts

- Conflicts: none identified in the provided inputs.
- Missing formal schema for the prompt envelope and work list ordering
  (depends on output format definitions in F-002).
- Missing definition of "fresh context" (process isolation, temp files, and
  environment cleanup between iterations).
- Missing explicit exit-signal format and detection rules for harness
  responses.
- Missing precedence rules between CLI flags and `.dun/config.yaml` for
  harness and automation defaults (depends on F-007).
- Missing credential handling and secret injection rules for harnesses
  (API keys, env vars, or config redaction).
- Dependencies: auto-discovery (F-001), output formats (F-002), automation
  policy (F-007), and doc drift inputs from reconciliation (F-006).

## Traceability

- Supports PRD goals for deterministic, agent-friendly output and fast local
  feedback loops.
- Supports PRD scope for doc-to-code reconciliation by enabling autonomous
  iteration once drift is detected.
