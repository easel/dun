---
dun:
  id: F-015
  depends_on:
    - helix.prd
  review:
    self_hash: 008dfe6a9c5e1cc46ed047edc19cf8970646ebd5be4dcbd63183b9ddb1080303
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-015 Exit Codes

## Summary

Define deterministic CLI exit codes so CI and automation can interpret Dun
outcomes reliably.

## Requirements

- Provide a stable, documented exit code list for all CLI commands.
- Use exit codes to distinguish success, usage errors, config errors, runtime
  errors, update failures, and quorum outcomes.
- Keep exit code selection deterministic for a given repo state and command.
- Document exit codes in `dun help` and in this spec.

## Inputs

- `internal/dun/exitcodes.go` for canonical constants.
- `cmd/dun/main.go` and `cmd/dun/review.go` for command behavior.
- `cmd/dun/main_test.go` for expected exit code usage.

## Gaps & Conflicts

- `ExitCheckFailed` is documented as "check failed", but `dun check` currently
  returns `ExitSuccess` even when individual checks report `Status: fail` or
  `Status: warn`; it only returns `ExitCheckFailed` on engine or encoding
  errors. Clarify whether exit code 1 should reflect check statuses.
- `ExitUpdateError` is defined and listed in help text, but update/version
  flows return `ExitRuntimeError` on update failures; align implementation or
  spec.
- Warning-only checks still map to exit code 0 because process exit codes do
  not reflect check results; confirm whether warnings should ever return
  non-zero.
- `dun loop` returns `ExitSuccess` when max iterations are reached, even if
  failing checks remain; policy for this outcome is not explicitly documented.

## Exit Codes

| Code | Constant | Description |
|------|----------|-------------|
| 0 | ExitSuccess | Command completed successfully. |
| 1 | ExitCheckFailed | Check command failed to execute or report results. |
| 2 | ExitConfigError | Configuration load or validation failed. |
| 3 | ExitRuntimeError | Runtime failure (I/O, missing files, harness errors). |
| 4 | ExitUsageError | Invalid flags, missing arguments, or unknown command. |
| 5 | ExitUpdateError | Update check or apply failed. |
| 6 | ExitQuorumConflict | Quorum could not reach consensus. |
| 7 | ExitQuorumAborted | Quorum was aborted by the user. |

## Behavior

- Exit codes are command-level signals; per-check pass/warn/fail statuses are
  reported in the output payloads.
- `dun check`, `dun list`, `dun explain`, and `dun respond` return
  `ExitSuccess` when they emit results, even if those results include failed
  checks.
- `dun loop` returns `ExitQuorumConflict` or `ExitQuorumAborted` when quorum
  execution fails, otherwise `ExitSuccess` when the loop completes or exits
  due to limits.
- `dun update` and `dun version --check` currently surface update failures as
  `ExitRuntimeError`.

## Acceptance Criteria

- `internal/dun/exitcodes.go` defines the full exit code set in this spec.
- `dun help` lists exit codes that match the constants.
- Usage, config, and runtime error paths return `ExitUsageError`,
  `ExitConfigError`, and `ExitRuntimeError` respectively.
- Quorum conflicts and aborts return `ExitQuorumConflict` and
  `ExitQuorumAborted`.

## Traceability

- Supports PRD goals for deterministic, agent-friendly output and CI
  integration with a stable exit-code policy.
