#!/bin/bash
set -e

VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0")}
PROJECT_NAME="bwenv"
RELEASE_DIR="releases"

echo "🚀 Building ${PROJECT_NAME} release ${VERSION}"

rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"

# Create a clean source archive (what gets distributed)
create_package() {
  local platform=$1
  local name="${PROJECT_NAME}-${VERSION}-${platform}"
  local staging="/tmp/${name}"

  rm -rf "$staging"
  mkdir -p "$staging/setup"

  cp README.md LICENSE "$staging/"
  cp setup/bitwarden_folders.sh "$staging/setup/"

  if [ "$platform" = "windows" ]; then
    cp setup/bwenv.bat "$staging/setup/"
    cp install.ps1 uninstall.ps1 "$staging/"
    (cd /tmp && zip -rq "${OLDPWD}/${RELEASE_DIR}/${name}.zip" "${name}/")
    echo "✅ ${name}.zip"
  else
    cp setup/bwenv "$staging/setup/"
    cp install.sh uninstall.sh Makefile "$staging/"
    chmod +x "$staging/setup/bwenv" "$staging/setup/bitwarden_folders.sh" "$staging/install.sh" "$staging/uninstall.sh"
    tar -czf "${RELEASE_DIR}/${name}.tar.gz" -C /tmp "${name}/"
    echo "✅ ${name}.tar.gz"
  fi

  rm -rf "$staging"
}

create_package "linux"
create_package "macos"
create_package "windows"

# Generate checksums
cd "${RELEASE_DIR}"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum *.tar.gz *.zip > checksums.txt
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 *.tar.gz *.zip > checksums.txt
fi
cd ..

echo ""
echo "🎉 Release ${VERSION} built:"
ls -lh "${RELEASE_DIR}/"
