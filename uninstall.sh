#!/usr/bin/env bash
set -euo pipefail

# bwenv uninstaller for Linux and macOS

INSTALL_LIB="${HOME}/.config/direnv/lib"
INSTALL_BIN="${HOME}/.local/bin"

echo "Uninstalling bwenv..."

rm -f "${INSTALL_LIB}/bitwarden_folders.sh" && echo "  [OK] Removed ${INSTALL_LIB}/bitwarden_folders.sh"
rm -f "${INSTALL_BIN}/bwenv" && echo "  [OK] Removed ${INSTALL_BIN}/bwenv"

echo ""
echo "bwenv uninstalled."
echo "   Note: direnv hooks in your shell config were not removed."
echo "   Remove them manually if no longer needed."
