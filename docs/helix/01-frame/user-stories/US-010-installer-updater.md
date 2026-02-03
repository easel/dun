---
dun:
  id: US-010
  depends_on:
  - F-019
---
# US-010: User-Friendly Installer and Self-Updater

As a developer, I want a simple way to install and update Dun so I can get
started quickly and stay current without manual steps.

## Acceptance Criteria

- One-line install command works on macOS and Linux.
- Install script detects architecture and downloads correct binary.
- `dun update` checks for new versions and self-updates.
- `dun version` shows current version and checks for updates.
- Installation adds dun to PATH or provides instructions.
- Updates preserve user configuration in `.dun/config.yaml`.
- Supports installation via common package managers (brew, apt, etc.).

## Example Usage

```bash
# One-line install
curl -sSL https://dun.dev/install.sh | sh

# Or via brew
brew install easel/tap/dun

# Check version and updates
dun version
# dun v0.2.0 (latest: v0.3.0 available)

# Self-update
dun update
# Updated dun from v0.2.0 to v0.3.0
```

## Design Notes

- Use GitHub releases for binary distribution
- Install script should be idempotent
- Consider goreleaser for cross-platform builds
- Version check should be non-blocking and cached
