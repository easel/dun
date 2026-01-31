# TP-010: Installer and Self-Updater Test Plan

**User Story**: US-010 - User-Friendly Installer and Self-Updater
**Status**: Planned (Not Yet Implemented)
**Date**: 2026-01-30
**Author**: QA Agent

## Overview

This test plan defines the verification strategy for the Dun installer and
self-updater feature. The feature is currently planned but not implemented.
These tests should be created alongside the implementation.

## Acceptance Criteria (from US-010)

1. One-line install command works on macOS and Linux.
2. Install script detects architecture and downloads correct binary.
3. `dun update` checks for new versions and self-updates.
4. `dun version` shows current version and checks for updates.
5. Installation adds dun to PATH or provides instructions.
6. Updates preserve user configuration in `.dun/config.yaml`.
7. Supports installation via common package managers (brew, apt, etc.).

## Test Categories

### TC-010.1: Install Script Tests

| ID | Test Case | Priority | Type |
|----|-----------|----------|------|
| TC-010.1.1 | Install script downloads and runs on macOS (Intel) | P0 | E2E |
| TC-010.1.2 | Install script downloads and runs on macOS (ARM64) | P0 | E2E |
| TC-010.1.3 | Install script downloads and runs on Linux (x86_64) | P0 | E2E |
| TC-010.1.4 | Install script downloads and runs on Linux (ARM64) | P1 | E2E |
| TC-010.1.5 | Install script detects unsupported architecture and exits with error | P1 | Integration |
| TC-010.1.6 | Install script is idempotent (running twice produces same result) | P0 | Integration |
| TC-010.1.7 | Install script adds dun to PATH or prints instructions | P0 | Integration |
| TC-010.1.8 | Install script handles network failure gracefully | P1 | Integration |
| TC-010.1.9 | Install script verifies checksum of downloaded binary | P1 | Integration |

### TC-010.2: Architecture Detection Tests

| ID | Test Case | Priority | Type |
|----|-----------|----------|------|
| TC-010.2.1 | Detects darwin/amd64 and downloads correct binary | P0 | Unit |
| TC-010.2.2 | Detects darwin/arm64 and downloads correct binary | P0 | Unit |
| TC-010.2.3 | Detects linux/amd64 and downloads correct binary | P0 | Unit |
| TC-010.2.4 | Detects linux/arm64 and downloads correct binary | P0 | Unit |
| TC-010.2.5 | Returns error for unsupported OS (Windows) | P1 | Unit |
| TC-010.2.6 | Returns error for unsupported architecture | P1 | Unit |

### TC-010.3: Version Command Tests

| ID | Test Case | Priority | Type |
|----|-----------|----------|------|
| TC-010.3.1 | `dun version` displays current version | P0 | Integration |
| TC-010.3.2 | `dun version` shows update available when newer version exists | P0 | Integration |
| TC-010.3.3 | `dun version` shows "up to date" when on latest version | P0 | Integration |
| TC-010.3.4 | Version check handles network timeout gracefully | P1 | Integration |
| TC-010.3.5 | Version check is non-blocking (completes within 2s) | P1 | Integration |
| TC-010.3.6 | Version check result is cached (subsequent calls use cache) | P2 | Integration |
| TC-010.3.7 | `dun version --json` outputs machine-readable format | P2 | Integration |

### TC-010.4: Update Command Tests

| ID | Test Case | Priority | Type |
|----|-----------|----------|------|
| TC-010.4.1 | `dun update` downloads and installs newer version | P0 | E2E |
| TC-010.4.2 | `dun update` reports "already up to date" when on latest | P0 | Integration |
| TC-010.4.3 | `dun update` preserves `.dun/config.yaml` after update | P0 | E2E |
| TC-010.4.4 | `dun update` handles permission errors (non-writable install dir) | P1 | Integration |
| TC-010.4.5 | `dun update` handles network failure with retry option | P1 | Integration |
| TC-010.4.6 | `dun update` verifies checksum before replacing binary | P1 | Integration |
| TC-010.4.7 | `dun update --dry-run` shows what would happen without updating | P2 | Integration |
| TC-010.4.8 | Update rollback on failure (keeps old binary if update fails) | P1 | E2E |

### TC-010.5: Configuration Preservation Tests

| ID | Test Case | Priority | Type |
|----|-----------|----------|------|
| TC-010.5.1 | Update preserves user-modified config values | P0 | Integration |
| TC-010.5.2 | Update adds new config keys with defaults | P1 | Integration |
| TC-010.5.3 | Update handles missing config file gracefully | P1 | Integration |
| TC-010.5.4 | Update handles corrupted config file with warning | P1 | Integration |

### TC-010.6: Package Manager Tests

| ID | Test Case | Priority | Type |
|----|-----------|----------|------|
| TC-010.6.1 | `brew install easel/tap/dun` installs correctly | P1 | E2E |
| TC-010.6.2 | `brew upgrade dun` updates to latest version | P1 | E2E |
| TC-010.6.3 | Homebrew formula includes correct dependencies | P2 | Contract |
| TC-010.6.4 | apt repository provides deb package (future) | P2 | E2E |

## Test Data Requirements

### Fixtures Needed

```
internal/testdata/
  installer/
    mock-releases/          # Mock GitHub release responses
      v0.2.0.json
      v0.3.0.json
    mock-binaries/          # Mock binary files for each platform
      dun-darwin-amd64
      dun-darwin-arm64
      dun-linux-amd64
      dun-linux-arm64
    configs/
      valid-config.yaml     # Sample user config
      corrupted-config.yaml # Malformed YAML
```

### Mock Services

- HTTP server to mock GitHub releases API
- Local file server for binary downloads during tests

## Test Implementation Notes

### Unit Tests (Go)

```go
// internal/installer/detect_test.go
func TestDetectArchitecture(t *testing.T) {
    tests := []struct {
        name     string
        goos     string
        goarch   string
        wantOS   string
        wantArch string
        wantErr  bool
    }{
        {"darwin amd64", "darwin", "amd64", "darwin", "amd64", false},
        {"darwin arm64", "darwin", "arm64", "darwin", "arm64", false},
        {"linux amd64", "linux", "amd64", "linux", "amd64", false},
        {"linux arm64", "linux", "arm64", "linux", "arm64", false},
        {"windows unsupported", "windows", "amd64", "", "", true},
    }
    // ... test implementation
}
```

### Integration Tests (Go)

```go
// internal/installer/update_test.go
func TestUpdatePreservesConfig(t *testing.T) {
    // Setup: Create temp dir with config
    // Action: Run update with mock server
    // Assert: Config values preserved
}
```

### E2E Tests (Shell Script)

```bash
# test/e2e/install_test.sh
test_install_script_idempotent() {
    # Run install twice
    # Verify same result
    # Verify binary works
}
```

## Success Criteria

| Metric | Target |
|--------|--------|
| Unit test coverage for installer package | 80% |
| All P0 tests passing | 100% |
| Install script works on CI (Linux) | Pass |
| Update preserves config in all scenarios | Pass |

## Dependencies

- GitHub releases API (or mock)
- goreleaser configuration for cross-platform builds
- Homebrew tap repository (`easel/tap`)

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Platform-specific behavior | High | Test on actual platforms in CI matrix |
| Network flakiness in tests | Medium | Use local mock server |
| Binary signing requirements | Medium | Document signing process |

## Notes

This feature depends on:
- Binary distribution infrastructure (GitHub releases)
- goreleaser configuration
- Homebrew tap setup

Tests should be implemented incrementally as the feature is built.
