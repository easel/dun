---
dun:
  id: TD-008
  depends_on:
  - US-008
---
# Technical Design: TD-008 Git Hygiene

## Story Reference

**User Story**: US-008 Keep Git Hygiene and Hook Checks
**Parent Feature**: F-005 Git Hygiene
**Solution Design**: SD-007 Git Hygiene

## Goals

- Detect dirty working trees and surface actionable issues.
- Run configured hook suites (lefthook or pre-commit) when present.
- Provide clear guidance when hook tooling is missing.

## Non-Goals

- Git history rewriting or commit signing.

## Technical Approach

### Implementation Strategy

- Use `git status --porcelain` to list dirty paths.
- Detect hook configuration files (lefthook, pre-commit).
- Execute hook runner when configured; warn if binaries are missing.

### Key Decisions

- Report each dirty path as a separate issue for clear remediation.
- Skip hook checks entirely if no hook configuration exists.

## Component Changes

### Components to Modify

- `internal/dun/git_checks.go`: implement git-status and git-hooks checks.
- `internal/plugins/builtin/git/plugin.yaml`: declare triggers and checks.

### New Components

- None.

## Interfaces and Config

- No new CLI flags; checks run as part of `dun check`.

## Data and State

- No persistent state required.

## Testing Approach

- Unit tests for parsing `git status` output.
- Integration tests for hook detection and missing tool warnings.

## Risks and Mitigations

- **Risk**: Hooks are slow or side-effectful. **Mitigation**: allow opt-out via
  config and respect automation mode.

## Rollout / Compatibility

- Backwards compatible; checks only activate in git repos.
