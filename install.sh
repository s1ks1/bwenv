#!/usr/bin/env bash
set -euo pipefail

# bwenv installer for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | bash

REPO="s1ks1/bwenv"
INSTALL_LIB="${HOME}/.config/direnv/lib"
INSTALL_BIN="${HOME}/.local/bin"
BRANCH="main"

info()  { echo "  ℹ️  $*"; }
ok()    { echo "  ✅ $*"; }
warn()  { echo "  ⚠️  $*"; }
err()   { echo "  ❌ $*" >&2; }

echo "🔧 Installing bwenv..."
echo ""

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  PLATFORM="linux" ;;
  Darwin*) PLATFORM="macos" ;;
  *)       err "Unsupported OS: $OS"; exit 1 ;;
esac
info "Detected platform: $PLATFORM"

# Check for required tools
check_dep() {
  if command -v "$1" >/dev/null 2>&1; then
    ok "$1 found"
  else
    warn "$1 not found — $2"
  fi
}

echo ""
echo "📋 Checking dependencies..."
check_dep bw "Install from https://bitwarden.com/help/cli/"
check_dep direnv "Install from https://direnv.net/"
check_dep jq "Install with: $([ "$PLATFORM" = "macos" ] && echo 'brew install jq' || echo 'sudo apt install jq')"

# Create directories
mkdir -p "$INSTALL_LIB"
mkdir -p "$INSTALL_BIN"

# Download files
echo ""
echo "📥 Downloading bwenv..."
BASE_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}"

curl -fsSL "${BASE_URL}/setup/bitwarden_folders.sh" -o "${INSTALL_LIB}/bitwarden_folders.sh"
chmod +x "${INSTALL_LIB}/bitwarden_folders.sh"
ok "Helper script installed to ${INSTALL_LIB}/bitwarden_folders.sh"

curl -fsSL "${BASE_URL}/setup/bwenv" -o "${INSTALL_BIN}/bwenv"
chmod +x "${INSTALL_BIN}/bwenv"
ok "CLI installed to ${INSTALL_BIN}/bwenv"

# Check PATH
echo ""
if echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_BIN"; then
  ok "${INSTALL_BIN} is in your PATH"
else
  warn "${INSTALL_BIN} is not in your PATH"
  SHELL_NAME="$(basename "$SHELL")"
  case "$SHELL_NAME" in
    bash) RC_FILE="$HOME/.bashrc" ;;
    zsh)  RC_FILE="$HOME/.zshrc" ;;
    fish) RC_FILE="$HOME/.config/fish/config.fish" ;;
    *)    RC_FILE="" ;;
  esac
  if [ -n "$RC_FILE" ] && [ -f "$RC_FILE" ]; then
    if ! grep -q "${INSTALL_BIN}" "$RC_FILE" 2>/dev/null; then
      if [ "$SHELL_NAME" = "fish" ]; then
        echo "set -gx PATH ${INSTALL_BIN} \$PATH" >> "$RC_FILE"
      else
        echo "export PATH=\"${INSTALL_BIN}:\$PATH\"" >> "$RC_FILE"
      fi
      ok "Added ${INSTALL_BIN} to ${RC_FILE}"
      info "Restart your terminal or run: source ${RC_FILE}"
    fi
  else
    info "Add this to your shell config: export PATH=\"${INSTALL_BIN}:\$PATH\""
  fi
fi

# Setup direnv hook
echo ""
echo "🔗 Checking direnv hook..."
if command -v direnv >/dev/null 2>&1; then
  SHELL_NAME="$(basename "$SHELL")"
  case "$SHELL_NAME" in
    bash)
      if [ -f "$HOME/.bashrc" ] && ! grep -q "direnv hook bash" "$HOME/.bashrc"; then
        echo 'eval "$(direnv hook bash)"' >> "$HOME/.bashrc"
        ok "Added direnv hook to ~/.bashrc"
      fi
      ;;
    zsh)
      if [ -f "$HOME/.zshrc" ] && ! grep -q "direnv hook zsh" "$HOME/.zshrc"; then
        echo 'eval "$(direnv hook zsh)"' >> "$HOME/.zshrc"
        ok "Added direnv hook to ~/.zshrc"
      fi
      ;;
    fish)
      if [ -f "$HOME/.config/fish/config.fish" ] && ! grep -q "direnv hook fish" "$HOME/.config/fish/config.fish"; then
        echo 'direnv hook fish | source' >> "$HOME/.config/fish/config.fish"
        ok "Added direnv hook to fish config"
      fi
      ;;
  esac
else
  warn "direnv not installed — hook setup skipped"
fi

echo ""
echo "🎉 bwenv installed successfully!"
echo "   Run 'bwenv test' to verify your setup."
echo ""
