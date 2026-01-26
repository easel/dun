# Solution Design: Go Quality Checks

## Goal

Provide deterministic test, coverage, and static analysis checks for Go repos
using a built-in Go plugin.

## Scope

- `go test ./...` with clear failure summaries.
- Coverage computation with a configurable minimum (default 100%).
- `go vet ./...` analysis.
- Optional `staticcheck ./...` when installed.

## Approach

1. **Discovery**: Activate when `go.mod` exists.
2. **Execution**:
   - `go-test`: run `go test ./...`.
   - `go-coverage`: run `go test ./... -coverprofile`, compute total coverage.
   - `go-vet`: run `go vet ./...`.
   - `go-staticcheck`: run when binary present; warn if missing.
3. **Reporting**: emit deterministic `signal`, `detail`, and `next`.

## Data Flow

1. Plugin discovery includes Go checks when `go.mod` exists.
2. Check runner executes commands in repo root.
3. Coverage parser extracts total percent and compares to threshold.
4. Reporter summarizes results.

## Open Questions

- Where should per-repo coverage thresholds live (config vs manifest)?
