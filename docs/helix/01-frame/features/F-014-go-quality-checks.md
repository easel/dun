---
dun:
  id: F-014
  depends_on:
    - helix.prd
  review:
    self_hash: 55b907d81942a3c0201d381c77e459657f8c2afb1810b6bd99561f161e7add9b
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-014 Go Quality Checks

## Summary

Provide built-in Go checks for tests, coverage, and static analysis so Dun can
enforce baseline quality in Go repositories.

## Requirements

- Detect Go repos via `go.mod`.
- Provide a built-in Go plugin with checks for `go-test`, `go-coverage`,
  `go-vet`, and `go-staticcheck`.
- Run `go test ./...` and report failures with actionable next steps.
- Compute total test coverage via `go test ./... -coverprofile` and
  `go tool cover -func`, then fail when below the configured threshold.
- Run `go vet ./...` and report failures.
- If `staticcheck` is available, run it; otherwise warn with a clear install hint.
- Allow coverage threshold overrides via `.dun/config.yaml` at
  `go.coverage_threshold` (default 80%).
- Keep output deterministic and LLM-friendly.

## Inputs

- Repository root file tree (for `go.mod` detection).
- Go toolchain (`go` binary) for tests, coverage, and vet.
- Optional `staticcheck` binary on `PATH`.
- `.dun/config.yaml` (for `go.coverage_threshold`).
- Built-in Go plugin manifest (`internal/plugins/builtin/go/plugin.yaml`).

## Gaps & Conflicts

- Timeout/cancellation behavior for long-running Go commands is not defined in
  the PRD or this spec.
- The spec does not define how coverage artifacts should be handled beyond
  local temporary files (naming, location, retention).
- Dependencies: this feature requires auto-discovery (F-001) and the plugin
  system (F-003) to activate and run the checks.
- No conflicts identified in the provided inputs.

## Detection

- Plugin activates when `go.mod` exists.
- `staticcheck` runs only when the binary is available on `PATH`.

## Check Behavior

- **go-test**:
  - **Fail** on non-zero exit with trimmed output in `detail`.
  - **Pass** on success.
- **go-coverage**:
  - Run `go test ./... -coverprofile` and parse total coverage from
    `go tool cover -func`.
  - **Fail** if `go test` fails, coverage parsing fails, or total coverage is
    below threshold.
  - **Pass** with `signal` showing the total coverage percentage.
- **go-vet**:
  - **Fail** on non-zero exit with trimmed output in `detail`.
  - **Pass** on success.
- **go-staticcheck**:
  - **Warn** if `staticcheck` is missing, with an install hint.
  - **Fail** when `staticcheck` runs but reports issues.
  - **Pass** on success.

## Output

- Each check emits a deterministic `signal`; `detail` and `next` are included
  when a failure or warning is actionable.
- Coverage failures include the current coverage percentage and the target
  threshold in `detail`.

## Non-Goals

- Replacing CI test suites.
- Advanced per-package coverage thresholds.

## Acceptance Criteria

- `dun check` runs Go checks when `go.mod` exists.
- `go-test` and `go-vet` fail with actionable output when their commands fail.
- Coverage check fails if total coverage is below the default threshold (80%)
  unless overridden in `.dun/config.yaml`:
  ```yaml
  go:
    coverage_threshold: 90
  ```
- Coverage check reports the current percentage and the target threshold.
- Staticcheck warns (not fails) when the tool is missing.
- Staticcheck fails when the tool runs and reports issues.

## Traceability

- Supports PRD goals for baseline Go quality checks (tests, coverage, static
  analysis) and deterministic, agent-friendly output.
