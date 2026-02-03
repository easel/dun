---
dun:
  id: TD-010
  depends_on:
  - US-010
---
# Technical Design: TD-010 Installer and Self-Updater

## Story Reference

**User Story**: US-010 Installer and Self-Updater
**Parent Feature**: F-019 Installer and Self-Updater
**Solution Design**: SD-010 Installer and Self-Updater

## Goals

- Provide a one-line install flow for macOS and Linux.
- Provide `dun update` to self-update from release artifacts.
- Preserve user configuration across updates.

## Non-Goals

- Windows installer support (future work).

## Technical Approach

### Implementation Strategy

- Distribute signed release artifacts with checksums.
- Install script detects OS/arch, downloads binary, verifies checksum.
- `dun update` fetches latest release metadata and replaces the binary
  atomically.

### Key Decisions

- Use GitHub Releases API for version discovery.
- Cache latest version metadata for a short interval to limit API calls.

## Component Changes

### Components to Modify

- `cmd/dun/main.go`: add `update` and `version --check` options.
- `internal/dun/update.go` (new): fetch release metadata and replace binary.
- `scripts/install.sh` (new): one-line installer script.

### New Components

- Release metadata parser and checksum verifier.

## Interfaces and Config

- CLI: `dun update`, `dun version --check`.
- Config: optional release channel (stable/beta).

## Data and State

- Cache file storing latest release metadata and timestamp.

## Testing Approach

- Unit tests for version parsing and checksum verification.
- Integration tests for update flow using mocked release server.

## Risks and Mitigations

- **Risk**: Corrupted downloads. **Mitigation**: checksum verification and
  rollback on failure.

## Rollout / Compatibility

- Backwards compatible; update path is opt-in.
