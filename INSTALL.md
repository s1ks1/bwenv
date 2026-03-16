# bwenv — Installation & Testing Guide

Complete instructions for installing bwenv and all prerequisites on **macOS**, **Linux**, and **Windows**, plus step-by-step testing workflows for **Bitwarden** and **1Password**.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Install bwenv](#install-bwenv)
  - [macOS](#macos)
  - [Linux (Ubuntu/Debian)](#linux-ubuntudebian)
  - [Linux (Fedora/RHEL)](#linux-fedorarhel)
  - [Linux (Arch)](#linux-arch)
  - [Windows](#windows)
  - [From Source (All Platforms)](#from-source-all-platforms)
- [Post-Install Setup](#post-install-setup)
  - [Configure direnv Hook](#configure-direnv-hook)
  - [Verify Installation](#verify-installation)
- [Testing with Bitwarden](#testing-with-bitwarden)
  - [Bitwarden Setup](#bitwarden-setup)
  - [Bitwarden Test: Interactive Mode](#bitwarden-test-interactive-mode)
  - [Bitwarden Test: Non-Interactive Mode](#bitwarden-test-non-interactive-mode)
  - [Bitwarden: CI/CD Usage](#bitwarden-cicd-usage)
- [Testing with 1Password](#testing-with-1password)
  - [1Password Setup](#1password-setup)
  - [1Password Test: Interactive Mode](#1password-test-interactive-mode)
  - [1Password Test: Non-Interactive Mode](#1password-test-non-interactive-mode)
  - [1Password: CI/CD Usage](#1password-cicd-usage)
- [Troubleshooting](#troubleshooting)
- [Uninstall](#uninstall)

---

## Prerequisites

bwenv requires **two** things:

| Tool | Why | Install link |
|------|-----|-------------|
| **direnv** | Automatically loads/unloads env vars when you `cd` into a directory | [direnv.net](https://direnv.net/) |
| **Password manager CLI** (at least one) | Fetches secrets from your vault | See below |

### Password Manager CLIs

| Provider | CLI | Install link |
|----------|-----|-------------|
| Bitwarden | `bw` | [bitwarden.com/help/cli](https://bitwarden.com/help/cli/) |
| 1Password | `op` | [developer.1password.com/docs/cli](https://developer.1password.com/docs/cli/) |

---

## Install bwenv

### Quick Install (Recommended)

**macOS / Linux** — one-line install via curl:

```bash
curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | sh
```

Or with wget:

```bash
wget -qO- https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | sh
```

**Windows** — one-line install via PowerShell:

```powershell
irm https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1 | iex
```

> **Note:** Both scripts auto-detect your OS and architecture, download the latest release, verify checksums, and install to `~/.local/bin`. You can customize the version and install directory:
>
> ```bash
> # macOS/Linux: custom version and directory
> BWENV_VERSION=v2.0.0 BWENV_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | sh
>
> # Windows: custom version and directory
> irm https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1 | iex -Version v2.0.0 -InstallDir C:\Tools
> ```

### macOS

```bash
# 1. Install direnv
brew install direnv

# 2. Install a password manager CLI (pick one or both)
brew install bitwarden-cli    # Bitwarden
brew install --cask 1password-cli  # 1Password

# 3. Install bwenv
brew tap s1ks1/bwenv
brew install bwenv
```

### Linux (Ubuntu/Debian)

```bash
# 1. Install direnv
sudo apt update && sudo apt install -y direnv

# 2. Install Bitwarden CLI
sudo snap install bw
# OR download from: https://bitwarden.com/help/cli/#download-and-install

# 2b. Install 1Password CLI (optional, instead of or in addition to Bitwarden)
# Add 1Password APT repo:
curl -sS https://downloads.1password.com/linux/keys/1password.asc | \
  sudo gpg --dearmor --output /usr/share/keyrings/1password-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/1password-archive-keyring.gpg] https://downloads.1password.com/linux/debian/$(dpkg --print-architecture) stable main" | \
  sudo tee /etc/apt/sources.list.d/1password.list
sudo apt update && sudo apt install -y 1password-cli

# 3. Install bwenv (via Go)
go install github.com/s1ks1/bwenv@latest

# OR download the binary directly:
# Visit https://github.com/s1ks1/bwenv/releases
# Download the linux-amd64 or linux-arm64 tarball
# Extract and move to a directory in your PATH:
tar xzf bwenv-*-linux-amd64.tar.gz
sudo mv bwenv-*-linux-amd64/bwenv /usr/local/bin/
```

### Linux (Fedora/RHEL)

```bash
# 1. Install direnv
sudo dnf install -y direnv

# 2. Install Bitwarden CLI
sudo snap install bw
# OR use npm: npm install -g @bitwarden/cli

# 2b. Install 1Password CLI (optional)
sudo rpm --import https://downloads.1password.com/linux/keys/1password.asc
sudo sh -c 'echo -e "[1password]\nname=1Password\nbaseurl=https://downloads.1password.com/linux/rpm/stable/\$basearch\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\ngpgkey=https://downloads.1password.com/linux/keys/1password.asc" > /etc/yum.repos.d/1password.repo'
sudo dnf install -y 1password-cli

# 3. Install bwenv
go install github.com/s1ks1/bwenv@latest
```

### Linux (Arch)

```bash
# 1. Install direnv
sudo pacman -S direnv

# 2. Install Bitwarden CLI (from AUR)
yay -S bitwarden-cli
# OR: paru -S bitwarden-cli

# 2b. Install 1Password CLI (optional, from AUR)
yay -S 1password-cli

# 3. Install bwenv
go install github.com/s1ks1/bwenv@latest
```

### Windows

```powershell
# Option A: Using Scoop (recommended)

# 1. Install direnv
scoop install direnv

# 2. Install a password manager CLI
scoop install bitwarden-cli    # Bitwarden
# OR
scoop install 1password-cli    # 1Password

# 3. Install bwenv
scoop bucket add bwenv https://github.com/s1ks1/scoop-bwenv
scoop install bwenv
```

```powershell
# Option B: Using Chocolatey

# 1. Install direnv
choco install direnv

# 2. Install Bitwarden CLI
choco install bitwarden-cli
# OR 1Password CLI — download from https://developer.1password.com/docs/cli/

# 3. Install bwenv via Go
go install github.com/s1ks1/bwenv@latest
```

```powershell
# Option C: Using winget

# 1. Install direnv
winget install direnv.direnv

# 2. Install CLIs
winget install Bitwarden.CLI
# OR
winget install AgileBits.1Password.CLI

# 3. Install bwenv via Go
go install github.com/s1ks1/bwenv@latest
```

### From Source (All Platforms)

```bash
# Requires Go 1.22+ installed
git clone https://github.com/s1ks1/bwenv.git
cd bwenv
make install    # Builds and installs to ~/.local/bin

# Verify
bwenv version
```

---

## Post-Install Setup

### Configure direnv Hook

direnv needs to be hooked into your shell. **This is a one-time setup.**

#### Bash

Add to `~/.bashrc`:

```bash
eval "$(direnv hook bash)"
```

#### Zsh

Add to `~/.zshrc`:

```bash
eval "$(direnv hook zsh)"
```

#### Fish

Add to `~/.config/fish/config.fish`:

```fish
direnv hook fish | source
```

#### PowerShell

Add to your PowerShell profile (`$PROFILE`):

```powershell
Invoke-Expression "$(direnv hook pwsh)"
```

After adding the hook, **restart your terminal** or source your shell config:

```bash
source ~/.zshrc   # or ~/.bashrc, etc.
```

### Verify Installation

```bash
# Run the status check to verify everything is properly configured
bwenv status
```

This will show:
- Whether direnv is installed and its hook is configured
- Which password manager CLIs are available
- Current session states
- Configuration preferences

---

## Testing with Bitwarden

### Bitwarden Setup

Before testing bwenv with Bitwarden, you need some test secrets in your vault.

**Step 1: Log in to Bitwarden CLI**

```bash
# First-time login (you'll be prompted for email + master password)
bw login

# If already logged in, just unlock
bw unlock
```

**Step 2: Create a test folder**

You can do this via the Bitwarden web vault or the CLI:

```bash
# Create a folder (via web vault is recommended)
# Go to vault.bitwarden.com → Folders → Create "bwenv-test"
```

**Step 3: Add test items with custom fields**

In the Bitwarden web vault or app:

1. Create a new **Secure Note** or **Login** item
2. Place it in the `bwenv-test` folder
3. Add **Custom Fields**:
   - Field name: `DB_HOST` → Value: `localhost`
   - Field name: `DB_PORT` → Value: `5432`
   - Field name: `DB_PASSWORD` → Value: `test-secret-123`
   - Field name: `API_KEY` → Value: `sk-test-key-abc`

> **Important:** bwenv reads **custom fields**, not the standard username/password fields. Each custom field becomes one environment variable.

**Step 4: Sync your vault**

```bash
bw sync
```

### Bitwarden Test: Interactive Mode

```bash
# Navigate to a test directory
mkdir -p ~/bwenv-test && cd ~/bwenv-test

# Run the full interactive flow
bwenv init
# → Select "Bitwarden" as the provider
# → Enter your master password when prompted
# → Select the "bwenv-test" folder
# → bwenv will preview variables, generate .envrc, and auto-approve it

# Trigger direnv to load the secrets
cd .

# Verify secrets are in the environment
echo $DB_HOST        # Should print: localhost
echo $DB_PORT        # Should print: 5432
echo $API_KEY        # Should print: sk-test-key-abc
env | grep DB_       # Should show DB_HOST, DB_PORT, DB_PASSWORD

# Check status
bwenv status

# Clean up when done
bwenv remove
cd ~
rm -rf ~/bwenv-test
```

### Bitwarden Test: Non-Interactive Mode

```bash
# Unlock Bitwarden and get session token
export BW_SESSION=$(bw unlock --raw)

# Export secrets directly (prints "export KEY=VALUE" lines)
bwenv export --provider bitwarden --folder "bwenv-test"

# Or load directly into your shell
eval "$(bwenv export --provider bitwarden --folder "bwenv-test")"

# Verify
echo $DB_HOST
echo $API_KEY

# Lock vault when done
bwenv logout
```

### Bitwarden: CI/CD Usage

```bash
# In CI, use BW_SESSION from environment
export BW_SESSION="${{ secrets.BW_SESSION }}"

# Load secrets for the deployment
eval "$(bwenv export --provider bitwarden --folder "Production")"

# Use secrets in your deployment
./deploy.sh  # $DB_URL, $API_KEY, etc. are available
```

---

## Testing with 1Password

### 1Password Setup

**Step 1: Install and configure the `op` CLI**

```bash
# Verify op is installed
op --version

# Sign in to your 1Password account
op signin
# On macOS/Windows with 1Password desktop app, this uses biometric auth
```

**Step 2: Create a test vault** (or use an existing one)

```bash
# List existing vaults
op vault list

# Or create a test vault via the 1Password app/web
# Create vault named "bwenv-test"
```

**Step 3: Add test items**

Using the 1Password app or CLI:

```bash
# Create a test item with fields
op item create \
  --category login \
  --title "Test Secrets" \
  --vault "bwenv-test" \
  --generate-password \
  username=testuser \
  'DB_HOST[text]=localhost' \
  'DB_PORT[text]=5432' \
  'API_KEY[text]=sk-test-key-abc'
```

Or manually in the 1Password app:

1. Go to the `bwenv-test` vault
2. Create a new item
3. Add fields with labels like `DB_HOST`, `DB_PORT`, `API_KEY`

> **Important:** bwenv reads item **fields**. The field label becomes the env var name, the field value becomes the env var value. Notes and OTP fields are skipped.

### 1Password Test: Interactive Mode

```bash
# Navigate to a test directory
mkdir -p ~/bwenv-test && cd ~/bwenv-test

# Run the full interactive flow
bwenv init
# → Select "1Password" as the provider
# → Authenticate via biometrics/system prompt (op CLI v2)
# → Select the "bwenv-test" vault
# → bwenv will preview variables, generate .envrc, and auto-approve it

# Trigger direnv to load the secrets
cd .

# Verify secrets are in the environment
echo $DB_HOST        # Should print: localhost
echo $DB_PORT        # Should print: 5432
echo $API_KEY        # Should print: sk-test-key-abc
env | grep DB_       # Should show DB_HOST, DB_PORT

# Check status
bwenv status

# Clean up when done
bwenv remove
cd ~
rm -rf ~/bwenv-test
```

### 1Password Test: Non-Interactive Mode

```bash
# Ensure you're signed in (op v2 uses system auth)
op signin

# Export secrets directly
bwenv export --provider 1password --folder "bwenv-test"

# Or load directly into your shell
eval "$(bwenv export --provider 1password --folder "bwenv-test")"

# Verify
echo $DB_HOST
echo $API_KEY

# Lock vault when done
bwenv logout
```

### 1Password: CI/CD Usage

```bash
# In CI, use a service account token
export OP_SERVICE_ACCOUNT_TOKEN="${{ secrets.OP_SERVICE_ACCOUNT_TOKEN }}"

# Load secrets for the deployment
eval "$(bwenv export --provider 1password --folder "Production")"

# Use secrets in your deployment
./deploy.sh  # $DB_URL, $API_KEY, etc. are available
```

---

## Troubleshooting

### Common Issues

#### "direnv: error .envrc is blocked"

```bash
bwenv allow    # Approve the .envrc file
```

#### "direnv: loading .envrc" messages appearing

These messages come from direnv's shell hook, not from bwenv itself.
bwenv automatically silences them by adding `DIRENV_LOG_FORMAT=""` to your shell RC file.

```bash
# If messages still appear, restart your shell:
exec $SHELL

# Or manually re-source your config:
source ~/.zshrc   # or ~/.bashrc, etc.

# You can also toggle the setting via:
bwenv config   # Toggle "Show Direnv Output" to OFF
```

#### Bitwarden: "Your vault is locked"

```bash
bwenv login               # Re-authenticate and update .envrc in one step
```

If `bwenv login` doesn't work (e.g. no `.envrc` exists yet):

```bash
bwenv logout              # Clear stale sessions
bw unlock                 # Unlock vault again
bwenv init                # Re-run setup to get a fresh session
```

#### Bitwarden: "Session key is invalid"

The BW_SESSION token has expired. Re-authenticate:

```bash
bwenv login               # Fastest way — re-auths and updates .envrc
```

#### 1Password: "not signed in"

```bash
bwenv login               # Re-authenticate and update .envrc
# Or manually:
op signin                 # Re-authenticate
bwenv init                # Re-run setup
```

#### No secrets found / 0 variables loaded

For **Bitwarden**: Ensure your items have **custom fields** (not just the standard username/password). Each custom field becomes one environment variable.

For **1Password**: Ensure your items have fields with **labels and values**. Fields without labels or with empty values are skipped. Notes and OTP fields are also skipped.

#### direnv not loading on terminal start

Make sure the direnv hook is in your shell RC file:

```bash
bwenv status   # Check the "Dependencies" section
```

#### bwenv not found

Make sure the install directory is in your `PATH`:

```bash
# If installed via go install:
export PATH="$HOME/go/bin:$PATH"

# If installed via make install:
export PATH="$HOME/.local/bin:$PATH"
```

### Full Diagnostic Check

Run the comprehensive status command to see everything at a glance:

```bash
bwenv status
```

This checks:
- `.envrc` presence and content
- direnv installation and hook
- Provider CLI availability
- Active sessions
- Relevant environment variables
- Config preferences

---

## Uninstall

### macOS (Homebrew)

```bash
brew uninstall bwenv
brew untap s1ks1/bwenv
```

### Windows (Scoop)

```powershell
scoop uninstall bwenv
scoop bucket rm bwenv
```

### Go install

```bash
rm -f $(go env GOPATH)/bin/bwenv
```

### From source

```bash
cd bwenv
make uninstall
```

### Clean up bwenv config

```bash
rm -rf ~/.config/bwenv
```

### Remove direnv silence line (optional)

If bwenv added `export DIRENV_LOG_FORMAT=""` to your shell RC, you can remove it:

```bash
# Check if it's there
grep DIRENV_LOG_FORMAT ~/.zshrc   # or ~/.bashrc

# Remove the line if you want direnv messages back
# Edit your shell RC file and remove the DIRENV_LOG_FORMAT line
```

### Remove .envrc from projects

In each project directory where bwenv was used:

```bash
bwenv remove    # or just: rm .envrc
```
