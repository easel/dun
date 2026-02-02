# Feature Spec: F-014 Go Quality Checks

## Summary

Provide built-in Go checks for tests, coverage, and static analysis so Dun can
enforce baseline quality in Go repositories.

## Requirements

- Detect Go repos via `go.mod`.
- Run `go test ./...` and report failures with actionable next steps.
- Compute total test coverage and fail when below the configured threshold.
- Run `go vet ./...` and report failures.
- If `staticcheck` is available, run it; otherwise warn with a clear install hint.
- Keep output deterministic and LLM-friendly.

## Detection

- Plugin activates when `go.mod` exists.
- `staticcheck` runs only when the binary is available on `PATH`.

## Check Behavior

- **go-test**: fail on non-zero exit, include failing package in `detail`.
- **go-coverage**: compute total coverage; fail when below threshold.
- **go-vet**: fail on non-zero exit with `detail` of vet output.
- **go-staticcheck**: warn if tool missing; fail if tool reports issues.

## Output

- Each check emits `signal`, `detail`, and `next` with a reproducible command.
- Coverage check includes current coverage and target threshold.

## Non-Goals

- Replacing CI test suites.
- Advanced per-package coverage thresholds.

## Acceptance Criteria

- `dun check` runs go checks when `go.mod` exists.
- Coverage check fails if total coverage is below the default threshold (80%) unless overridden in `.dun/config.yaml`:
  ```yaml
  go:
    coverage_threshold: 90
  ```
- Staticcheck is skipped or warned when the tool is missing.
