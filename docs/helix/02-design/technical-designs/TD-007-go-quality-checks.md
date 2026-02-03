---
dun:
  id: TD-007
  depends_on:
  - US-007
---
# Technical Design: TD-007 Go Quality Checks

## Story Reference

**User Story**: US-007 Go Quality Checks
**Parent Feature**: F-014 Go Quality Checks
**Solution Design**: SD-006 Go Quality Checks

## Goals

- Provide deterministic Go checks: tests, coverage, vet, staticcheck.
- Emit actionable failure output for each check.
- Keep checks fast and opt-in based on repo signals.

## Non-Goals

- Dependency vulnerability scanning (covered by security plugins).

## Technical Approach

### Implementation Strategy

- Activate Go checks when `go.mod` is present.
- Implement checks as command runners with consistent error handling.
- Parse coverage output and compare to configurable thresholds.

### Key Decisions

- Use `go test ./...` for test and coverage to align with Go tooling.
- Treat missing `staticcheck` as a warning, not a hard failure.

## Component Changes

### Components to Modify

- `internal/dun/go_checks.go`: implement go-test, go-coverage, go-vet,
  go-staticcheck checks.
- `internal/plugins/builtin/go/plugin.yaml`: define Go checks and triggers.

### New Components

- None.

## Interfaces and Config

- Config: `go.coverage_threshold` in `.dun/config.yaml`.

## Data and State

- Coverage profile written to a temp file or repo root per configuration.

## Testing Approach

- Unit tests for coverage parsing and command error handling.
- Integration tests for plan activation when `go.mod` exists.

## Risks and Mitigations

- **Risk**: Long-running tests. **Mitigation**: allow timeouts and skip
  commands on CI constraints.

## Rollout / Compatibility

- Backwards compatible; checks only activate for Go repos.
