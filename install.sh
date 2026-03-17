#!/bin/sh
# =============================================================================
# bwenv — Cross-platform installer for macOS and Linux
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | sh
#   wget -qO- https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | sh
#
# Options (via environment variables):
#   BWENV_VERSION   Specific version to install (default: latest)
#   BWENV_DIR       Installation directory (default: ~/.local/bin)
#
# Supports: macOS (amd64, arm64), Linux (amd64, arm64)
# =============================================================================
set -e

# -- Configuration --
GITHUB_REPO="s1ks1/bwenv"
INSTALL_DIR="${BWENV_DIR:-$HOME/.local/bin}"
TMP_DIR=""

# -- Colors --
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info()    { printf "${BLUE}${BOLD}[info]${NC}  %s\n" "$1"; }
success() { printf "${GREEN}${BOLD}[ok]${NC}    %s\n" "$1"; }
warn()    { printf "${YELLOW}${BOLD}[warn]${NC}  %s\n" "$1"; }
error()   { printf "${RED}${BOLD}[error]${NC} %s\n" "$1" >&2; exit 1; }

# -- Cleanup on exit --
cleanup() {
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}
trap cleanup EXIT

# -- Detect OS and architecture --
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      error "Unsupported operating system: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)   ARCH="amd64" ;;
        aarch64|arm64)  ARCH="arm64" ;;
        *)              error "Unsupported architecture: $ARCH" ;;
    esac

    PLATFORM="${OS}-${ARCH}"
}

# -- Detect download tool --
detect_downloader() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# -- Download a URL to a file --
download() {
    url="$1"
    output="$2"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url" -o "$output"
    else
        wget -qO "$output" "$url"
    fi
}

# -- Fetch content from a URL --
fetch() {
    url="$1"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url"
    else
        wget -qO- "$url"
    fi
}

# -- Get the latest release version from GitHub --
get_latest_version() {
    # Use the GitHub API to get the latest release tag.
    LATEST_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    VERSION=$(fetch "$LATEST_URL" 2>/dev/null | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Set BWENV_VERSION manually."
    fi
}

# -- Verify checksum (if checksums.txt is available) --
verify_checksum() {
    archive_path="$1"
    archive_name="$2"
    version="$3"

    CHECKSUMS_URL="https://github.com/${GITHUB_REPO}/releases/download/${version}/checksums.txt"

    info "Verifying checksum..."
    CHECKSUMS=$(fetch "$CHECKSUMS_URL" 2>/dev/null || true)

    if [ -z "$CHECKSUMS" ]; then
        warn "Checksums not available — skipping verification"
        return
    fi

    EXPECTED=$(echo "$CHECKSUMS" | grep "$archive_name" | awk '{print $1}')
    if [ -z "$EXPECTED" ]; then
        warn "No checksum found for $archive_name — skipping verification"
        return
    fi

    # Compute actual checksum.
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL=$(sha256sum "$archive_path" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL=$(shasum -a 256 "$archive_path" | awk '{print $1}')
    else
        warn "No sha256sum or shasum found — skipping checksum verification"
        return
    fi

    if [ "$EXPECTED" = "$ACTUAL" ]; then
        success "Checksum verified"
    else
        error "Checksum mismatch!\n  Expected: $EXPECTED\n  Actual:   $ACTUAL"
    fi
}

# -- Main --
main() {
    printf "\n"
    printf "  ${BOLD}${BLUE}bwenv installer${NC}\n"
    printf "  ─────────────────────────────\n\n"

    detect_platform
    detect_downloader
    info "Detected platform: ${PLATFORM}"

    # Determine version.
    if [ -n "$BWENV_VERSION" ]; then
        VERSION="$BWENV_VERSION"
        info "Using specified version: ${VERSION}"
    else
        info "Fetching latest release..."
        get_latest_version
        info "Latest version: ${VERSION}"
    fi

    # Strip 'v' prefix for the archive naming convention.
    VERSION_NUM="${VERSION#v}"

    # Construct download URL.
    ARCHIVE_NAME="bwenv-${VERSION}-${PLATFORM}.tar.gz"
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

    # Create temp directory.
    TMP_DIR="$(mktemp -d)"

    # Download the archive.
    info "Downloading ${ARCHIVE_NAME}..."
    download "$DOWNLOAD_URL" "${TMP_DIR}/${ARCHIVE_NAME}" || \
        error "Download failed. Check that version ${VERSION} exists for ${PLATFORM}.\n  URL: ${DOWNLOAD_URL}"

    success "Downloaded successfully"

    # Verify checksum.
    verify_checksum "${TMP_DIR}/${ARCHIVE_NAME}" "$ARCHIVE_NAME" "$VERSION"

    # Extract the archive.
    info "Extracting..."
    tar xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}"

    # Find the binary (it's inside a subdirectory).
    BINARY=$(find "${TMP_DIR}" -name "bwenv" -type f | head -1)
    if [ -z "$BINARY" ]; then
        error "Could not find bwenv binary in the archive"
    fi

    # Install to the target directory.
    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    cp "$BINARY" "${INSTALL_DIR}/bwenv"
    chmod +x "${INSTALL_DIR}/bwenv"

    success "Installed bwenv ${VERSION} to ${INSTALL_DIR}/bwenv"

    # Check if install directory is in PATH.
    printf "\n"
    case ":$PATH:" in
        *":${INSTALL_DIR}:"*)
            success "${INSTALL_DIR} is in your PATH"
            ;;
        *)
            warn "${INSTALL_DIR} is NOT in your PATH"
            printf "\n"
            info "Add it to your shell config:"
            printf "    ${BOLD}export PATH=\"%s:\$PATH\"${NC}\n" "$INSTALL_DIR"
            printf "\n"

            # Detect shell and suggest the right RC file.
            SHELL_NAME="$(basename "${SHELL:-/bin/sh}")"
            case "$SHELL_NAME" in
                zsh)  RC_FILE="~/.zshrc" ;;
                bash)
                    if [ "$(uname -s)" = "Darwin" ]; then
                        RC_FILE="~/.bash_profile"
                    else
                        RC_FILE="~/.bashrc"
                    fi
                    ;;
                fish) RC_FILE="~/.config/fish/config.fish" ;;
                *)    RC_FILE="~/.profile" ;;
            esac
            info "Or add permanently to ${RC_FILE}:"
            printf "    ${BOLD}echo 'export PATH=\"%s:\$PATH\"' >> %s${NC}\n" "$INSTALL_DIR" "$RC_FILE"
            ;;
    esac

    # Verify the installation.
    printf "\n"
    if command -v bwenv >/dev/null 2>&1; then
        INSTALLED_VERSION="$(bwenv version 2>/dev/null | head -1 || echo "unknown")"
        success "bwenv is ready! Run 'bwenv status' to verify your setup."
    else
        info "Installation complete. Restart your shell or source your config, then run:"
        printf "    ${BOLD}bwenv status${NC}\n"
    fi

    printf "\n"
}

main "$@"
