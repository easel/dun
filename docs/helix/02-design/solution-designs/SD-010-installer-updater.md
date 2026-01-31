# SD-010: Installer and Self-Updater

## Overview

This document describes the architecture and implementation plan for a user-friendly
installer and self-updater for Dun. Users will be able to install via a one-line
curl command or Homebrew, check their version, and self-update without manual steps.

**User Story**: US-010
**Test Plan**: TP-010
**Status**: Planned

## Architecture

### Component Overview

```
+------------------+     +-------------------+     +------------------+
|   install.sh     |---->|  GitHub Releases  |<----|  goreleaser      |
| (shell script)   |     |  (binary hosting) |     |  (build system)  |
+------------------+     +-------------------+     +------------------+
                               ^
                               |
+------------------+     +-------------------+     +------------------+
|   dun version    |---->|  Version Checker  |---->|  Update Cache    |
|   (CLI command)  |     |  (GitHub API)     |     |  (local file)    |
+------------------+     +-------------------+     +------------------+
                               |
                               v
+------------------+     +-------------------+
|   dun update     |---->|  Self-Updater     |
|   (CLI command)  |     |  (binary replace) |
+------------------+     +-------------------+
```

### Data Flow

1. **Installation Flow**:
   - User runs `curl -sSL https://dun.dev/install.sh | sh`
   - Script detects OS (darwin/linux) and architecture (amd64/arm64)
   - Script downloads binary from GitHub Releases
   - Script verifies checksum
   - Script installs to `/usr/local/bin` or `$HOME/.local/bin`
   - Script provides PATH instructions if needed

2. **Version Check Flow**:
   - `dun version` calls GitHub API (releases/latest)
   - Response is cached locally for 1 hour
   - Compares current embedded version with latest tag
   - Displays version info and update availability

3. **Update Flow**:
   - `dun update` checks for newer version
   - Downloads new binary to temp location
   - Verifies checksum before replacing
   - Atomically replaces current binary
   - Verifies new binary runs correctly
   - Rolls back on failure

## New Files to Create

### 1. Version Package
**Path**: `/home/erik/gt/dun/crew/oscar/internal/version/version.go`

```go
package version

// Version is set at build time via ldflags
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)

type Info struct {
    Version   string `json:"version"`
    Commit    string `json:"commit"`
    BuildDate string `json:"build_date"`
    GoVersion string `json:"go_version"`
    Platform  string `json:"platform"`
}

func Get() Info { ... }
func String() string { ... }
```

### 2. Update Package
**Path**: `/home/erik/gt/dun/crew/oscar/internal/update/update.go`

```go
package update

type Updater struct {
    CurrentVersion string
    RepoOwner      string
    RepoName       string
    BinaryName     string
}

type Release struct {
    TagName     string
    PublishedAt time.Time
    Assets      []Asset
}

type Asset struct {
    Name        string
    DownloadURL string
    Size        int64
}

func (u *Updater) CheckForUpdate() (*Release, bool, error) { ... }
func (u *Updater) DownloadRelease(release *Release) (string, error) { ... }
func (u *Updater) ApplyUpdate(binaryPath string) error { ... }
func (u *Updater) Rollback() error { ... }
```

### 3. Update Cache
**Path**: `/home/erik/gt/dun/crew/oscar/internal/update/cache.go`

```go
package update

type Cache struct {
    LastCheck    time.Time `json:"last_check"`
    LatestVersion string   `json:"latest_version"`
    UpdateAvail  bool      `json:"update_available"`
}

func (c *Cache) Load() error { ... }
func (c *Cache) Save() error { ... }
func (c *Cache) IsStale() bool { ... }
```

### 4. Install Script
**Path**: `/home/erik/gt/dun/crew/oscar/scripts/install.sh`

Shell script for one-line installation (detailed below).

### 5. goreleaser Configuration
**Path**: `/home/erik/gt/dun/crew/oscar/.goreleaser.yaml`

Build configuration for cross-platform releases.

### 6. Homebrew Formula
**Path**: Separate repository `easel/homebrew-tap/Formula/dun.rb`

Ruby formula for `brew install easel/tap/dun`.

### 7. Test Files
**Paths**:
- `/home/erik/gt/dun/crew/oscar/internal/version/version_test.go`
- `/home/erik/gt/dun/crew/oscar/internal/update/update_test.go`
- `/home/erik/gt/dun/crew/oscar/internal/update/cache_test.go`
- `/home/erik/gt/dun/crew/oscar/internal/testdata/installer/` (fixtures)

## Changes to Existing Files

### 1. cmd/dun/main.go

Add two new commands: `version` and `update`.

```go
// Add to run() switch statement:
case "version":
    return runVersion(args[1:], stdout, stderr)
case "update":
    return runUpdate(args[1:], stdout, stderr)

// Add new functions:
func runVersion(args []string, stdout io.Writer, stderr io.Writer) int {
    fs := flag.NewFlagSet("version", flag.ContinueOnError)
    jsonOutput := fs.Bool("json", false, "output as JSON")
    checkOnly := fs.Bool("check", false, "check for updates without displaying version")
    if err := fs.Parse(args); err != nil {
        return dun.ExitUsageError
    }
    // ... implementation
}

func runUpdate(args []string, stdout io.Writer, stderr io.Writer) int {
    fs := flag.NewFlagSet("update", flag.ContinueOnError)
    dryRun := fs.Bool("dry-run", false, "show what would happen without updating")
    force := fs.Bool("force", false, "force update even if on latest version")
    if err := fs.Parse(args); err != nil {
        return dun.ExitUsageError
    }
    // ... implementation
}
```

Update help text to include new commands:
```
COMMANDS:
  ...
  version    Show version info and check for updates
  update     Self-update to the latest version
```

### 2. internal/dun/exitcodes.go

Add new exit code:
```go
const (
    // ... existing codes
    ExitUpdateError = 5 // Update failed
)
```

### 3. Makefile (if exists) or go.mod build tags

Add build-time version injection:
```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/easel/dun/internal/version.Version=$(VERSION) \
           -X github.com/easel/dun/internal/version.Commit=$(COMMIT) \
           -X github.com/easel/dun/internal/version.BuildDate=$(BUILD_DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o dun ./cmd/dun
```

## External Dependencies

### 1. goreleaser

goreleaser handles cross-compilation and release automation.

**Installation**: `brew install goreleaser` or via GitHub Action

**Purpose**:
- Build binaries for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- Generate checksums (SHA256)
- Create GitHub releases automatically
- Generate Homebrew formula updates

### 2. GitHub Releases API

**Endpoints used**:
- `GET /repos/{owner}/{repo}/releases/latest` - Check latest version
- Asset download URLs from release response

**Rate Limiting**:
- Unauthenticated: 60 requests/hour
- Cache responses locally to minimize API calls

### 3. No new Go dependencies required

The implementation uses only stdlib:
- `net/http` for GitHub API calls
- `crypto/sha256` for checksum verification
- `os` for file operations
- `runtime` for platform detection

## Install Script Design

**Location**: `/home/erik/gt/dun/crew/oscar/scripts/install.sh`

```bash
#!/bin/sh
set -e

# Configuration
REPO="easel/dun"
BINARY_NAME="dun"
INSTALL_DIR="${DUN_INSTALL_DIR:-/usr/local/bin}"

# Detect platform
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        *)
            echo "Unsupported OS: $OS"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)
            echo "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest release tag
get_latest_version() {
    curl -sL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install() {
    PLATFORM=$(detect_platform)
    VERSION=$(get_latest_version)

    if [ -z "$VERSION" ]; then
        echo "Failed to get latest version"
        exit 1
    fi

    echo "Installing dun ${VERSION} for ${PLATFORM}..."

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/dun_${PLATFORM}.tar.gz"
    CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    TMPDIR=$(mktemp -d)
    trap "rm -rf $TMPDIR" EXIT

    # Download binary and checksums
    curl -sL "$DOWNLOAD_URL" -o "$TMPDIR/dun.tar.gz"
    curl -sL "$CHECKSUM_URL" -o "$TMPDIR/checksums.txt"

    # Verify checksum
    cd "$TMPDIR"
    EXPECTED=$(grep "dun_${PLATFORM}.tar.gz" checksums.txt | awk '{print $1}')
    ACTUAL=$(sha256sum dun.tar.gz | awk '{print $1}')

    if [ "$EXPECTED" != "$ACTUAL" ]; then
        echo "Checksum verification failed!"
        exit 1
    fi

    # Extract and install
    tar -xzf dun.tar.gz

    # Check if install dir is writable
    if [ -w "$INSTALL_DIR" ]; then
        mv dun "$INSTALL_DIR/"
    else
        echo "Installing to $INSTALL_DIR requires elevated privileges..."
        sudo mv dun "$INSTALL_DIR/"
    fi

    echo "dun installed successfully to $INSTALL_DIR/dun"

    # Check if in PATH
    if ! command -v dun >/dev/null 2>&1; then
        echo ""
        echo "Note: $INSTALL_DIR may not be in your PATH."
        echo "Add the following to your shell profile:"
        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

install
```

**Key features**:
- Idempotent (safe to run multiple times)
- Platform detection (darwin/linux, amd64/arm64)
- Checksum verification
- Graceful handling of permissions
- Clear error messages
- PATH guidance when needed

## Version Command Design

**Usage**:
```
dun version [options]

Options:
  --json     Output as JSON
  --check    Only check for updates (no version display)
```

**Output formats**:

Standard output:
```
dun v0.2.0 (commit: abc1234, built: 2026-01-31)
Update available: v0.3.0
```

JSON output (`--json`):
```json
{
  "version": "0.2.0",
  "commit": "abc1234",
  "build_date": "2026-01-31T10:00:00Z",
  "go_version": "go1.25.5",
  "platform": "darwin/arm64",
  "latest_version": "0.3.0",
  "update_available": true
}
```

**Behavior**:
1. Display embedded version info
2. Asynchronously check GitHub for updates (with 2s timeout)
3. Cache result for 1 hour in `~/.dun/update-cache.json`
4. If update available, show notification
5. Exit 0 regardless of update availability

## Update Command Design

**Usage**:
```
dun update [options]

Options:
  --dry-run  Show what would happen without updating
  --force    Force update even if already on latest
```

**Output**:
```
Checking for updates...
Current version: v0.2.0
Latest version: v0.3.0

Downloading dun v0.3.0...
Verifying checksum...
Installing update...
Updated successfully from v0.2.0 to v0.3.0
```

**Algorithm**:
```
1. Check current version against GitHub releases
2. If no update available and not --force:
   Print "Already up to date" and exit 0
3. Determine current binary path via os.Executable()
4. Download new binary to temp file
5. Verify checksum of downloaded binary
6. If --dry-run: print plan and exit 0
7. Rename current binary to .old backup
8. Move new binary to install location
9. Verify new binary runs (dun version)
10. If verification fails: rollback (.old -> current)
11. Remove .old backup on success
12. Print success message
```

**Error handling**:
- Network failures: clear error message, suggest retry
- Permission errors: suggest running with sudo or changing install dir
- Checksum mismatch: abort with clear error
- Verification failure: automatic rollback
- Config preservation: updates only replace binary, not ~/.dun/

## Package Manager Integration

### Homebrew Tap

**Repository**: `easel/homebrew-tap`

**Formula location**: `Formula/dun.rb`

```ruby
class Dun < Formula
  desc "Development quality checks and autonomous iteration"
  homepage "https://github.com/easel/dun"
  version "0.3.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/easel/dun/releases/download/v0.3.0/dun_darwin_amd64.tar.gz"
      sha256 "abc123..."
    end
    on_arm do
      url "https://github.com/easel/dun/releases/download/v0.3.0/dun_darwin_arm64.tar.gz"
      sha256 "def456..."
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/easel/dun/releases/download/v0.3.0/dun_linux_amd64.tar.gz"
      sha256 "ghi789..."
    end
    on_arm do
      url "https://github.com/easel/dun/releases/download/v0.3.0/dun_linux_arm64.tar.gz"
      sha256 "jkl012..."
    end
  end

  def install
    bin.install "dun"
  end

  test do
    assert_match "dun v#{version}", shell_output("#{bin}/dun version")
  end
end
```

**goreleaser integration**: goreleaser can auto-update the Homebrew formula on release.

### Future: APT Repository

For Debian/Ubuntu users, a future phase could add:
- GitHub Actions to build `.deb` packages
- Hosting on GitHub Pages or packagecloud.io
- Instructions for adding the repository

This is lower priority than curl and Homebrew.

## Implementation Phases

### Phase 1: Version Infrastructure (1-2 days)
**Tasks**:
1. Create `internal/version/version.go` with version struct
2. Create `internal/version/version_test.go`
3. Add `runVersion` function to `cmd/dun/main.go`
4. Add build-time ldflags to inject version
5. Update help text

**Acceptance**: `dun version` shows embedded version info

### Phase 2: GitHub Integration (1-2 days)
**Tasks**:
1. Create `internal/update/update.go` with GitHub API client
2. Create `internal/update/cache.go` for response caching
3. Integrate version check into `dun version` command
4. Add `--json` and `--check` flags
5. Write unit tests with mock HTTP server

**Acceptance**: `dun version` shows update availability (cached)

### Phase 3: Self-Update (2-3 days)
**Tasks**:
1. Implement binary download in update package
2. Implement checksum verification
3. Implement atomic binary replacement with rollback
4. Add `runUpdate` function to `cmd/dun/main.go`
5. Add `--dry-run` and `--force` flags
6. Write integration tests

**Acceptance**: `dun update` successfully updates the binary

### Phase 4: Install Script (1 day)
**Tasks**:
1. Create `scripts/install.sh`
2. Test on macOS (Intel + ARM)
3. Test on Linux (x86_64 + ARM64)
4. Host script (GitHub raw or dun.dev domain)
5. Document installation in README

**Acceptance**: One-line install works on all platforms

### Phase 5: Release Automation (1-2 days)
**Tasks**:
1. Create `.goreleaser.yaml`
2. Set up GitHub Actions workflow for releases
3. Configure checksum generation
4. Test release process with a pre-release tag

**Acceptance**: `git tag v0.x.0 && git push --tags` triggers automated release

### Phase 6: Homebrew (1 day)
**Tasks**:
1. Create `easel/homebrew-tap` repository
2. Add `Formula/dun.rb`
3. Configure goreleaser to update formula
4. Test `brew install easel/tap/dun`

**Acceptance**: Homebrew installation and upgrade work

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Binary replacement fails mid-update | Low | High | Atomic rename with rollback; verify before delete |
| GitHub API rate limiting | Medium | Low | Cache responses for 1 hour; graceful degradation |
| Checksum mismatch (network corruption) | Low | Medium | Fail loudly, suggest retry |
| User lacks write permissions | Medium | Low | Detect early, provide clear sudo/path guidance |
| Platform detection fails | Low | Medium | Explicit error messages; fallback prompts |
| Homebrew formula out of sync | Medium | Low | goreleaser auto-update; CI verification |
| Self-update creates infinite loop | Low | High | Version comparison prevents re-update |
| Old binary left behind on rollback | Low | Low | Cleanup .old files on successful update |
| Config files corrupted during update | Low | High | Update only touches binary, never config |

## Security Considerations

1. **Checksum verification**: All downloads verified against SHA256 checksums
2. **HTTPS only**: All downloads and API calls use HTTPS
3. **No code execution from network**: Downloaded binary is executed only after verification
4. **Minimal permissions**: Binary replacement uses atomic rename, not temp execution
5. **No elevated privileges required**: Defaults to user-writable location
6. **Signed binaries (future)**: Consider code signing for macOS Gatekeeper

## Testing Strategy

Per TP-010, tests cover:
- Architecture detection (unit tests)
- Version display and parsing (unit tests)
- Update check with mock server (integration tests)
- Binary replacement with rollback (integration tests)
- Install script on multiple platforms (E2E in CI matrix)
- Config preservation across updates (E2E tests)

## Open Questions

1. Should `dun update` support updating to a specific version (`dun update v0.2.0`)?
2. Should there be a `--no-verify` flag to skip checksum (not recommended)?
3. Should update notifications be suppressible via config?
4. Should install.sh support installing to custom locations via env var?
5. Should we support Windows in the future?

## References

- US-010: User Story for installer/updater
- TP-010: Test Plan for installer/updater
- goreleaser documentation: https://goreleaser.com/
- GitHub Releases API: https://docs.github.com/en/rest/releases
