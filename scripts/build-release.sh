#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get version from git tag or default
VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0")}
PROJECT_NAME="bwenv"
BUILD_DIR="dist"
RELEASE_DIR="releases"

echo -e "${BLUE}🚀 Building ${PROJECT_NAME} release ${VERSION}${NC}"

# Clean and create directories
rm -rf "${BUILD_DIR}" "${RELEASE_DIR}"
mkdir -p "${BUILD_DIR}" "${RELEASE_DIR}"

# Function to create platform-specific package
create_package() {
    local platform=$1
    local ext=$2
    local package_name="${PROJECT_NAME}-${VERSION}-${platform}"
    local package_dir="${BUILD_DIR}/${package_name}"

    echo -e "${YELLOW}📦 Creating package for ${platform}...${NC}"

    # Create package directory
    mkdir -p "${package_dir}"

    # Copy main files
    cp README.md "${package_dir}/"
    cp Makefile "${package_dir}/"
    cp LICENSE "${package_dir}/"
    cp -r assets "${package_dir}/"
    cp -r setup "${package_dir}/"

    # Create installation instructions for the platform
    cat > "${package_dir}/INSTALL.md" << EOF
# ${PROJECT_NAME} Installation

## Quick Install

\`\`\`bash
make install
EOF

    if [ "${platform}" != "windows" ]; then
        cat >> "${package_dir}/INSTALL.md" << 'EOF'
make setup-path  # Add ~/.local/bin to PATH (recommended)
```

## Manual Installation

1. Copy `setup/bitwarden_folders.sh` to `~/.config/direnv/lib/`
2. Copy `setup/bwenv` to `~/.local/bin/` and make it executable
3. Add `~/.local/bin` to your PATH
4. Ensure direnv is installed and configured

## Dependencies

- [Bitwarden CLI](https://bitwarden.com/help/cli/)
- [direnv](https://direnv.net/)
- [jq](https://stedolan.github.io/jq/)

## Verification

Run `bwenv test` to verify installation.
EOF
    else
        cat >> "${package_dir}/INSTALL.md" << 'EOF'
```

## Manual Installation (Windows)

1. Copy `setup/bitwarden_folders.sh` to `%USERPROFILE%\.config\direnv\lib\`
2. Copy `setup/bwenv.bat` to `%USERPROFILE%\.local\bin\`
3. Add `%USERPROFILE%\.local\bin` to your PATH environment variable

## Dependencies

- [Bitwarden CLI](https://bitwarden.com/help/cli/)
- [direnv](https://direnv.net/)
- [jq](https://stedolan.github.io/jq/) or jq for Windows

## Verification

Run `bwenv test` to verify installation.
EOF
    fi

    # Create archive
    cd "${BUILD_DIR}"
    if [ "${platform}" = "windows" ]; then
        zip -r "../${RELEASE_DIR}/${package_name}.zip" "${package_name}/" > /dev/null
        echo -e "${GREEN}✅ Created ${package_name}.zip${NC}"
    else
        tar -czf "../${RELEASE_DIR}/${package_name}.tar.gz" "${package_name}/"
        echo -e "${GREEN}✅ Created ${package_name}.tar.gz${NC}"
    fi
    cd ..
}

# Create packages for different platforms
create_package "linux" "tar.gz"
create_package "macos" "tar.gz"
create_package "windows" "zip"

# Create checksums
cd "${RELEASE_DIR}"
echo -e "${YELLOW}🔐 Generating checksums...${NC}"
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum * > checksums.txt
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 * > checksums.txt
else
    echo -e "${RED}⚠️ No checksum utility found (sha256sum or shasum)${NC}"
fi
cd ..

# Create release notes
cat > "${RELEASE_DIR}/RELEASE_NOTES.md" << EOF
# ${PROJECT_NAME} ${VERSION}

## 🚀 Features

- Interactive Bitwarden folder selection
- Automatic \`.envrc\` generation for direnv
- Secure session management with Bitwarden CLI
- Cross-platform support (Linux, macOS, Windows)
- Debug modes for troubleshooting
- Easy install/uninstall scripts

## 📦 Installation

### Quick Install
\`\`\`bash
# Download and extract the appropriate package for your platform
# Then run:
make install
make setup-path  # Linux/macOS only
\`\`\`

### Dependencies
- [Bitwarden CLI](https://bitwarden.com/help/cli/)
- [direnv](https://direnv.net/)
- [jq](https://stedolan.github.io/jq/)

## 🧪 Usage

\`\`\`bash
bwenv init          # Initialize secrets from Bitwarden folder
bwenv interactive   # Interactive folder selection
bwenv test          # Test installation and dependencies
bwenv remove        # Remove secrets from project
\`\`\`

## 🔐 Security

All secrets are loaded directly from your Bitwarden vault into your local environment. No secrets are stored in plaintext files or transmitted over the network beyond the standard Bitwarden CLI operations.

## 🐛 Verification

After installation, run \`bwenv test\` to verify all dependencies and configuration.
EOF

echo ""
echo -e "${GREEN}🎉 Release ${VERSION} built successfully!${NC}"
echo -e "${BLUE}📁 Files created in ${RELEASE_DIR}/:${NC}"
ls -la "${RELEASE_DIR}/"

echo ""
echo -e "${YELLOW}📋 Next steps:${NC}"
echo "1. Review the generated packages in ${RELEASE_DIR}/"
echo "2. Test installation on target platforms"
echo "3. Create GitHub release with these assets"
echo "4. Upload the packages and checksums to GitHub"
