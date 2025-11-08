<div align="center">
  <img src="./assets/Logo.svg" alt="Bwenv Logo" width="120"/>
  <h1>🔐 bwenv: Bitwarden + direnv Helper</h1>
  <p><em>Effortlessly sync secrets from Bitwarden folders into your shell environment with <b>direnv</b> magic!</em></p>
</div>

---

## 🚀 Overview

**bwenv** is a CLI tool that bridges your Bitwarden vault and your shell environment using [direnv](https://direnv.net/). It lets you securely load secrets from any Bitwarden folder directly into your `.envrc`, making secret management for development and deployment a breeze.

---

## ✨ Features

- **Interactive folder selection**: Pick Bitwarden folders interactively or by name
- **Automatic `.envrc` generation**: Instantly create a ready-to-use `.envrc` for direnv
- **Secure session management**: Handles Bitwarden session unlocking for you
- **Debug mode**: See exactly which secrets are loaded
- **Easy install/uninstall**: One command setup & cleanup

---

## 🛠️ Installation

### Quick Install

```bash
make install
```

This will:

- Copy helper scripts to `~/.config/direnv/lib`
- Install the `bwenv` CLI to `~/.local/bin` (Unix/macOS) or `%USERPROFILE%\.local\bin` (Windows)
- Make everything executable

> **Note:** Requires [Bitwarden CLI](https://bitwarden.com/help/cli/) and [jq](https://stedolan.github.io/jq/) installed.

### Platform-Specific Setup

#### Linux/macOS

After installation, you may need to add `~/.local/bin` to your PATH:

```bash
# Automatic setup (recommended)
make setup-path

# Manual setup - Add to your shell config (~/.bashrc, ~/.zshrc, etc.)
export PATH="$HOME/.local/bin:$PATH"
```

Then restart your terminal or run:
```bash
source ~/.bashrc  # or ~/.zshrc
```

#### Windows

On Windows, make sure `%USERPROFILE%\.local\bin` is in your PATH environment variable:

```cmd
# Add to PATH (PowerShell as Administrator)
$env:PATH += ";$env:USERPROFILE\.local\bin"
[Environment]::SetEnvironmentVariable("PATH", $env:PATH, [EnvironmentVariableTarget]::User)
```

### Available Commands

```bash
make help        # Show all available commands
make install     # Install bwenv CLI
make setup-path  # Add ~/.local/bin to PATH automatically (Linux/macOS)
make uninstall   # Remove bwenv CLI
```

---

## ⚡ Usage

### 1. Initialize secrets for your project

```bash
bwenv init                    # Default: show steps, hide secrets
bwenv --debug=2 init         # Full debug: show steps and secrets
bwenv --quiet init           # Quiet mode: minimal output
```

- Prompts for Bitwarden folder name
- Unlocks your vault and generates `.envrc`
- Run `direnv allow` to activate secrets

### 2. Interactive folder selection

```bash
bwenv interactive            # Default: show steps, hide secrets
bwenv --debug interactive    # Full debug: show steps and secrets
```

- Lists all Bitwarden folders
- Select by number for quick setup

### 3. Test installation

```bash
bwenv test
```

- Checks all dependencies and configuration
- Verifies direnv hook setup
- Tests Bitwarden session validity

### 4. Remove secrets

```bash
bwenv remove
```

- Deletes `.envrc` from your project

### Debug Options

- `--quiet, -q`: No debug output (BWENV_DEBUG=0)
- `--debug=1`: Show steps only, hide secrets (default)
- `--debug=2` or `--debug`: Show steps and secrets (full debug)

---

## 🧩 How It Works

- **Helper script**: Loads all custom fields from items in the selected Bitwarden folder as environment variables
- **Smart debugging**:
  - `BWENV_DEBUG=0`: Silent mode
  - `BWENV_DEBUG=1`: Shows processing steps, hides secret values (default)
  - `BWENV_DEBUG=2`: Shows processing steps and actual secret values
- **Session**: Uses `BW_SESSION` for secure access to your vault
- **Auto-setup**: Automatically configures direnv hooks in your shell

---

## 📦 Example Workflow

```bash
# Install and setup
make install
make setup-path              # Add ~/.local/bin to PATH (Linux/macOS)

# Test installation
bwenv test

# Initialize secrets
bwenv init                   # Manual folder entry
# or
bwenv interactive           # Pick from list

# Allow direnv to load secrets (with debug info)
direnv allow

# Verify secrets are loaded
echo $YOUR_SECRET_VAR

# Remove secrets when done
bwenv remove
```

### Debug Examples

```bash
# Quiet mode (no debug output)
bwenv --quiet init

# Default mode (show steps, hide secrets)
bwenv init

# Full debug (show steps and secrets)
bwenv --debug=2 interactive

# Test with different debug levels
BWENV_DEBUG=1 direnv allow   # Steps only
BWENV_DEBUG=2 direnv allow   # Full debug
BWENV_DEBUG=0 direnv allow   # Silent
```

---

## 🖼️ Screenshots

### Interactive Selection
<div align="center">
  <img src="./assets/bwenv-interactive.png" alt="Interactive folder selection" width="600"/>
</div>

### Init Command
<div align="center">
  <img src="./assets/bwenv-init.png" alt="Init command output" width="600"/>
</div>

---

## 📝 License

MIT License. See [LICENSE](LICENSE) for details.

---

## 🤝 Contributing

Pull requests welcome! For major changes, open an issue first to discuss what you’d like to change.

---

<div align="center">
  <b>Made with ❤️ for easy development</b>
  <b>Check out my profile: https://buymeacoffee.com/s1ks1</b>
</div>
