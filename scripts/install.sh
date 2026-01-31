#!/bin/sh
# install.sh - Installer for dun binary
# Downloads and installs the latest release from GitHub
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/easel/dun/main/scripts/install.sh | sh
#
# Environment variables:
#   DUN_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#   DUN_VERSION     - Specific version to install (default: latest)

set -e

# Configuration
REPO="easel/dun"
BINARY_NAME="dun"
INSTALL_DIR="${DUN_INSTALL_DIR:-/usr/local/bin}"
GITHUB_API="https://api.github.com"
GITHUB_RELEASES="https://github.com/${REPO}/releases/download"

# Colors for output (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Print functions
info() {
    printf "${BLUE}==>${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1" >&2
}

error() {
    printf "${RED}Error:${NC} %s\n" "$1" >&2
    exit 1
}

# Detect platform and architecture
# Sets: OS, ARCH
detect_platform() {
    # Detect OS
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        mingw*|msys*|cygwin*)
            error "Windows is not supported. Please use WSL or download manually."
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac

    # Detect architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        armv7l|armv6l)
            error "32-bit ARM is not supported. Please use arm64."
            ;;
        i386|i686)
            error "32-bit x86 is not supported. Please use a 64-bit system."
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    info "Detected platform: ${OS}/${ARCH}"
}

# Check for required commands
check_dependencies() {
    for cmd in curl tar; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            error "Required command not found: $cmd"
        fi
    done

    # Check for sha256sum or shasum
    if command -v sha256sum >/dev/null 2>&1; then
        SHA_CMD="sha256sum"
    elif command -v shasum >/dev/null 2>&1; then
        SHA_CMD="shasum -a 256"
    else
        error "Neither sha256sum nor shasum found. Cannot verify checksum."
    fi
}

# Get the latest version from GitHub API
# Sets: VERSION
get_latest_version() {
    if [ -n "${DUN_VERSION:-}" ]; then
        VERSION="$DUN_VERSION"
        info "Using specified version: $VERSION"
        return
    fi

    info "Fetching latest version..."

    # Fetch latest release tag from GitHub API
    LATEST_URL="${GITHUB_API}/repos/${REPO}/releases/latest"

    # Try to get the tag_name from the API response
    VERSION=$(curl -fsSL "$LATEST_URL" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/' | head -1)

    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version from GitHub. Check your network connection or specify DUN_VERSION."
    fi

    info "Latest version: $VERSION"
}

# Download and verify the binary
# Arguments: $1 = temp directory
download_and_verify() {
    TMPDIR="$1"

    # Construct download URLs
    # Version tag typically starts with 'v', archive name doesn't include it
    VERSION_NUM="${VERSION#v}"
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    CHECKSUMS_NAME="checksums.txt"

    ARCHIVE_URL="${GITHUB_RELEASES}/${VERSION}/${ARCHIVE_NAME}"
    CHECKSUMS_URL="${GITHUB_RELEASES}/${VERSION}/${CHECKSUMS_NAME}"

    info "Downloading ${ARCHIVE_NAME}..."
    if ! curl -fsSL -o "${TMPDIR}/${ARCHIVE_NAME}" "$ARCHIVE_URL"; then
        error "Failed to download binary archive from: $ARCHIVE_URL"
    fi

    info "Downloading checksums..."
    if ! curl -fsSL -o "${TMPDIR}/${CHECKSUMS_NAME}" "$CHECKSUMS_URL"; then
        error "Failed to download checksums from: $CHECKSUMS_URL"
    fi

    info "Verifying checksum..."
    cd "$TMPDIR"

    # Extract the expected checksum for our archive
    EXPECTED_CHECKSUM=$(grep "${ARCHIVE_NAME}" "${CHECKSUMS_NAME}" | awk '{print $1}')
    if [ -z "$EXPECTED_CHECKSUM" ]; then
        error "Checksum not found for ${ARCHIVE_NAME} in checksums.txt"
    fi

    # Calculate actual checksum
    ACTUAL_CHECKSUM=$($SHA_CMD "${ARCHIVE_NAME}" | awk '{print $1}')

    if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
        error "Checksum verification failed!
Expected: $EXPECTED_CHECKSUM
Actual:   $ACTUAL_CHECKSUM
The downloaded file may be corrupted or tampered with."
    fi

    success "Checksum verified"
}

# Extract and install the binary
# Arguments: $1 = temp directory
install_binary() {
    TMPDIR="$1"
    VERSION_NUM="${VERSION#v}"
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"

    info "Extracting archive..."
    cd "$TMPDIR"
    tar -xzf "${ARCHIVE_NAME}"

    # Verify binary exists after extraction
    if [ ! -f "${BINARY_NAME}" ]; then
        error "Binary not found in archive after extraction"
    fi

    # Make binary executable
    chmod +x "${BINARY_NAME}"

    # Check if we need sudo
    NEED_SUDO=""
    if [ ! -d "$INSTALL_DIR" ]; then
        # Try to create directory
        if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            NEED_SUDO="yes"
        fi
    elif [ ! -w "$INSTALL_DIR" ]; then
        NEED_SUDO="yes"
    fi

    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

    if [ -n "$NEED_SUDO" ]; then
        if ! command -v sudo >/dev/null 2>&1; then
            error "Cannot write to ${INSTALL_DIR} and sudo is not available.
Please run as root or set DUN_INSTALL_DIR to a writable directory:
  DUN_INSTALL_DIR=~/.local/bin sh -c '\$(curl -fsSL ...)'"
        fi

        warn "Requesting sudo to install to ${INSTALL_DIR}"
        sudo mkdir -p "$INSTALL_DIR"
        sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        mkdir -p "$INSTALL_DIR"
        mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    success "Installed ${BINARY_NAME} ${VERSION} to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Print post-installation instructions
print_instructions() {
    # Check if binary is in PATH
    if ! command -v "${BINARY_NAME}" >/dev/null 2>&1; then
        echo ""
        warn "${BINARY_NAME} is not in your PATH"
        echo ""
        echo "Add the following to your shell profile (.bashrc, .zshrc, etc.):"
        echo ""
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        echo "Then reload your shell or run:"
        echo ""
        echo "  source ~/.bashrc  # or ~/.zshrc"
        echo ""
    fi

    echo ""
    success "Installation complete!"
    echo ""
    echo "Get started with:"
    echo "  ${BINARY_NAME} help        # Show help"
    echo "  ${BINARY_NAME} install     # Set up dun in your project"
    echo "  ${BINARY_NAME} check       # Run quality checks"
    echo ""
}

# Main installation flow
main() {
    echo ""
    echo "Installing ${BINARY_NAME}..."
    echo ""

    # Pre-flight checks
    check_dependencies
    detect_platform
    get_latest_version

    # Create temporary directory with cleanup trap
    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT INT TERM

    # Download, verify, and install
    download_and_verify "$TMPDIR"
    install_binary "$TMPDIR"
    print_instructions
}

# Run main
main
