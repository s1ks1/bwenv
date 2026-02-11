#!/usr/bin/env bash
set -euo pipefail

# bwenv installer for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | bash

REPO="s1ks1/bwenv"
INSTALL_LIB="${HOME}/.config/direnv/lib"
INSTALL_BIN="${HOME}/.local/bin"
BRANCH="main"

info()  { echo "  [INFO] $*"; }
ok()    { echo "  [OK]   $*"; }
warn()  { echo "  [WARN] $*"; }

echo ""
echo "Installing bwenv..."
echo ""

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  PLATFORM="linux" ;;
  Darwin*) PLATFORM="macos" ;;
  *)       echo "  [ERROR] Unsupported OS: $OS" >&2; exit 1 ;;
esac
info "Platform: $PLATFORM"

# Check / auto-install dependencies
echo ""
echo "Checking dependencies..."

if command -v bw >/dev/null 2>&1; then
  ok "bw (Bitwarden CLI)"
else
  warn "bw not found - install from https://bitwarden.com/help/cli/"
fi

if command -v direnv >/dev/null 2>&1; then
  ok "direnv"
else
  warn "direnv not found - install from https://direnv.net/"
fi

if command -v jq >/dev/null 2>&1; then
  ok "jq"
else
  echo "  [....] jq not found - installing..."
  if [ "$PLATFORM" = "macos" ]; then
    if command -v brew >/dev/null 2>&1; then
      brew install jq >/dev/null 2>&1 && ok "jq installed via Homebrew" || warn "Failed to install jq"
    else
      warn "Install jq manually: brew install jq"
    fi
  else
    if command -v apt-get >/dev/null 2>&1; then
      sudo apt-get install -y jq >/dev/null 2>&1 && ok "jq installed via apt" || warn "Failed to install jq"
    elif command -v dnf >/dev/null 2>&1; then
      sudo dnf install -y jq >/dev/null 2>&1 && ok "jq installed via dnf" || warn "Failed to install jq"
    elif command -v pacman >/dev/null 2>&1; then
      sudo pacman -S --noconfirm jq >/dev/null 2>&1 && ok "jq installed via pacman" || warn "Failed to install jq"
    else
      warn "Install jq manually: https://stedolan.github.io/jq/"
    fi
  fi
fi

# Create directories
mkdir -p "$INSTALL_LIB"
mkdir -p "$INSTALL_BIN"

# Download files
echo ""
echo "Downloading bwenv..."
BASE_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}"

curl -fsSL "${BASE_URL}/setup/bitwarden_folders.sh" -o "${INSTALL_LIB}/bitwarden_folders.sh"
chmod +x "${INSTALL_LIB}/bitwarden_folders.sh"
ok "Helper script -> ${INSTALL_LIB}/bitwarden_folders.sh"

curl -fsSL "${BASE_URL}/setup/bwenv" -o "${INSTALL_BIN}/bwenv"
chmod +x "${INSTALL_BIN}/bwenv"
ok "CLI -> ${INSTALL_BIN}/bwenv"

# Check PATH
echo ""
if echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_BIN"; then
  ok "${INSTALL_BIN} is in PATH"
else
  warn "${INSTALL_BIN} is not in PATH"
  SHELL_NAME="$(basename "${SHELL:-bash}")"
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
      ok "Added to ${RC_FILE}"
      info "Restart your terminal or run: source ${RC_FILE}"
    fi
  else
    info "Add to your shell config: export PATH=\"${INSTALL_BIN}:\$PATH\""
  fi
fi

# Setup direnv hook
echo ""
if command -v direnv >/dev/null 2>&1; then
  SHELL_NAME="$(basename "${SHELL:-bash}")"
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
      mkdir -p "$HOME/.config/fish"
      if [ -f "$HOME/.config/fish/config.fish" ] && ! grep -q "direnv hook fish" "$HOME/.config/fish/config.fish"; then
        echo 'direnv hook fish | source' >> "$HOME/.config/fish/config.fish"
        ok "Added direnv hook to fish config"
      fi
      ;;
  esac
fi

echo ""
echo "bwenv installed successfully!"
echo "  Run 'bwenv test' to verify your setup."
echo ""
