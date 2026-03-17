<div align="center">
  <img src="./assets/Logo.svg" alt="bwenv Logo" width="120"/>
  <h1>🔐 bwenv</h1>
  <p><strong>Sync secrets from your password manager into your shell environment — beautifully.</strong></p>
  <p>
    <a href="#-installation"><img src="https://img.shields.io/badge/install-homebrew%20%7C%20scoop%20%7C%20apt%20%7C%20go-blue" alt="Install"/></a>
    <a href="https://github.com/s1ks1/bwenv/releases"><img src="https://img.shields.io/github/v/release/s1ks1/bwenv?style=flat&color=green" alt="Release"/></a>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-purple" alt="License"/></a>
    <a href="https://github.com/s1ks1/bwenv/actions"><img src="https://img.shields.io/github/actions/workflow/status/s1ks1/bwenv/release.yml?label=build" alt="Build"/></a>
    <a href="https://goreportcard.com/report/github.com/s1ks1/bwenv"><img src="https://goreportcard.com/badge/github.com/s1ks1/bwenv" alt="Go Report Card"/></a>
  </p>
</div>

---

>**_NOTE_**: Version v2 will be a complete rewrite in a new language, the current solution is not the best for cross platform installations, it also creates many obstacles for development and is not the best looking. As a lover of beautiful terminal applications, I wanted bwenv to become an aesthetically pleasing application even though it is a very simple purpose. With that, the next version goes to Go + Bubble Tea + Lipgloss + Make. The old version still remains in releases with all its advantages and disadvantages and will not be further developed.

## 🚀 Overview

**bwenv** is a cross-platform CLI tool that bridges your password manager and your shell environment using [direnv](https://direnv.net/). It lets you load secrets from **Bitwarden** or **1Password** directly into your project's environment variables — no manual copy-pasting, no secrets in `.env` files committed to git.

Built with [Go](https://go.dev/), [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lipgloss](https://github.com/charmbracelet/lipgloss) for a fast, beautiful, and truly cross-platform experience.

### Why bwenv?

Managing secrets across projects is painful. `.env` files get committed by accident, tokens expire and break your workflow, and switching between projects means manual copy-pasting. **bwenv** solves this by fetching secrets directly from your vault — live, per-directory, automatically.

The original bwenv was built with Bash scripts, which worked — but had constant cross-platform issues. This Go rewrite compiles to a **single static binary** for every platform, with zero runtime dependencies (beyond your password manager's CLI).

---

## ✨ Features

- **🔑 Multi-provider support** — Works with Bitwarden (`bw` CLI) and 1Password (`op` CLI)
- **🎨 Beautiful TUI** — Interactive provider and folder selection with arrow keys, search, and filtering
- **📁 Automatic `.envrc` generation** — Creates direnv-compatible files that auto-load your secrets
- **🖥️ True cross-platform** — Single binary for Linux, macOS, and Windows (amd64 + arm64)
- **🔍 Smart diagnostics** — `bwenv status` checks every dependency, session, and config
- **⚙️ Configurable UI** — Toggle emoji, direnv output, export summaries via `bwenv config`
- **🔑 Quick re-auth** — Session expired? `bwenv login` re-authenticates and updates your `.envrc` in one step
- **🔒 Secure logout** — Lock vaults and terminate sessions with `bwenv logout`
- **⚡ Zero runtime dependencies** — Just the Go binary + your password manager CLI + direnv
- **📦 Easy installation** — Homebrew, Scoop, APT/DNF, `go install`, or direct download

---

## 📦 Prerequisites

| Tool | Required? | Description |
|------|-----------|-------------|
| [direnv](https://direnv.net/) | **Yes** | Automatically loads/unloads environment variables from `.envrc` files |
| [Bitwarden CLI](https://bitwarden.com/help/cli/) | One of these | Access your Bitwarden vault from the terminal (`bw`) |
| [1Password CLI](https://developer.1password.com/docs/cli/) | One of these | Access your 1Password vaults from the terminal (`op`) |

> You need **at least one** password manager CLI installed. bwenv will detect what's available and let you choose.

---

## 🛠️ Installation

### Homebrew (macOS)

```bash
brew tap s1ks1/bwenv
brew install --cask bwenv
```

### Scoop (Windows)

```powershell
scoop bucket add bwenv https://github.com/s1ks1/scoop-bwenv
scoop install bwenv
```

### Linux (DEB — Debian / Ubuntu)

Download the `.deb` package from the [latest release](https://github.com/s1ks1/bwenv/releases/latest):

```bash
# Download (replace VERSION and ARCH as needed)
curl -LO https://github.com/s1ks1/bwenv/releases/latest/download/bwenv_VERSION_amd64.deb

# Install
sudo dpkg -i bwenv_*_amd64.deb
```

### Linux (RPM — Fedora / RHEL / openSUSE)

```bash
# Download (replace VERSION and ARCH as needed)
curl -LO https://github.com/s1ks1/bwenv/releases/latest/download/bwenv_VERSION_amd64.rpm

# Install
sudo rpm -i bwenv_*_amd64.rpm
```

### Go Install

```bash
go install github.com/s1ks1/bwenv@latest
```

### Quick Install Script

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | sh
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1 | iex
```

### From Source

```bash
git clone https://github.com/s1ks1/bwenv.git
cd bwenv
make install
```

### Direct Download

Download the latest binary for your platform from the [Releases](https://github.com/s1ks1/bwenv/releases) page, extract it, and place `bwenv` somewhere in your `PATH`.

### Verify Installation

```bash
bwenv status
```

> For detailed installation instructions on all platforms, including testing workflows for Bitwarden and 1Password, see [INSTALL.md](INSTALL.md).

---

## ⚡ Usage

### 1. Interactive Setup

```bash
bwenv init
```

This launches a full interactive TUI flow:

1. **Select a provider** — Choose between Bitwarden, 1Password (or whichever CLIs you have installed)
2. **Authenticate** — Unlock your vault or sign in (master password, biometrics, etc.)
3. **Pick a folder** — Browse, search, and select the folder/vault containing your secrets
4. **Generate `.envrc`** — A direnv-compatible file is created in the current directory

Then just:

```bash
cd .    # Trigger direnv to load secrets
```

Your secrets are now loaded as environment variables every time you `cd` into this directory! 🎉

### 2. Non-Interactive Export

For CI/CD pipelines, scripts, or advanced usage, you can export secrets directly:

```bash
# Output "export KEY=VALUE" lines to stdout
bwenv export --provider bitwarden --folder "MySecrets"

# Use with eval to set variables in the current shell
eval "$(bwenv export --provider bitwarden --folder "MySecrets")"

# Works with 1Password too
eval "$(bwenv export --provider 1password --folder "Production")"
```

### 3. Configure Preferences

```bash
bwenv config
```

Opens an interactive settings editor where you can toggle:

| Setting | Default | Description |
|---------|---------|-------------|
| **Show Emoji** | ON | Display emoji icons in output (turn off for cleaner text-only output) |
| **Show Direnv Output** | OFF | Show/hide direnv's own loading/unloading messages |
| **Show Export Summary** | ON | Show the boxed summary when secrets are loaded via direnv |
| **Auto Sync** | ON | Automatically sync the vault before fetching secrets (Bitwarden) |

Settings are persisted to `~/.config/bwenv/config.json`.

### 4. Re-authenticate (Session Expired)

```bash
bwenv login
```

If your vault session has expired, `bwenv login` will:
- Detect which provider is configured in your `.envrc`
- Re-authenticate with that provider (unlock/sign in)
- Update the session token in your `.envrc`
- Auto-approve the updated `.envrc` via direnv

This is much faster than running `bwenv init` again — it skips provider and folder selection entirely.

> **Alias:** `bwenv auth` works too.

### 5. Lock Vaults / Logout

```bash
bwenv logout
```

Terminates all active provider sessions for security:
- **Bitwarden** — runs `bw lock` to lock the vault
- **1Password** — runs `op signout` to end the session
- Shows any lingering session environment variables and how to clear them

Use this when you're done working with secrets or stepping away from your machine.

### 6. Status & Diagnostics

```bash
bwenv status
```

Shows a comprehensive overview of your current bwenv state:
- Current directory and `.envrc` info (provider, folder)
- direnv installation and hook status
- Provider availability and active sessions
- Relevant environment variables (masked for security)
- Current config preferences

### 7. Remove Secrets

```bash
bwenv remove
```

Deletes the `.envrc` file from the current directory.

### 8. Version

```bash
bwenv version
```

---

## 🧩 How It Works

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  bwenv init  │────▸│ Provider CLI │────▸│   .envrc     │
│  (TUI flow)  │     │ (bw / op)    │     │  (generated) │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                  │
                                                  ▼
                                          ┌──────────────┐
                                          │   direnv     │
                                          │  (auto-load) │
                                          └──────┬───────┘
                                                  │
                                                  ▼
                                          ┌──────────────┐
                                          │ Environment  │
                                          │  Variables   │
                                          │  $API_KEY    │
                                          │  $DB_URL     │
                                          │  $SECRET     │
                                          └──────────────┘
```

1. **`bwenv init`** walks you through an interactive setup — pick your provider and folder
2. It generates an `.envrc` file that contains a single `eval` call to `bwenv export`
3. When direnv loads the `.envrc`, it runs `bwenv export` which fetches fresh secrets from your vault
4. Each secret's custom fields (Bitwarden) or item fields (1Password) are exported as environment variables

**No secrets are stored on disk** (except session tokens which expire). The `.envrc` fetches secrets live from your vault each time direnv loads it.

---

## 📁 Project Structure

```
bwenv/
├── main.go                          # Entry point and CLI routing
├── INSTALL.md                       # Detailed install & testing guide
├── internal/
│   ├── provider/
│   │   ├── provider.go              # Provider interface and registry
│   │   ├── bitwarden.go             # Bitwarden (bw CLI) implementation
│   │   └── onepassword.go           # 1Password (op CLI) implementation
│   ├── ui/
│   │   ├── styles.go                # Lipgloss color palette and shared styles
│   │   ├── output.go                # Styled print helpers (success, error, etc.)
│   │   ├── provider_picker.go       # Bubble Tea model for provider selection
│   │   ├── folder_picker.go         # Bubble Tea model for folder selection
│   │   ├── init_flow.go             # Orchestrates the full init TUI flow
│   │   ├── login_flow.go            # Re-authentication flow for expired sessions
│   │   ├── config_flow.go           # Interactive config editor TUI
│   │   ├── logout_flow.go           # Vault locking and session termination
│   │   └── status_flow.go           # Status overview & diagnostics
│   ├── envrc/
│   │   └── envrc.go                 # .envrc generation, export, allow/disallow
│   ├── config/
│   │   └── config.go                # Persistent user preferences (~/.config/bwenv/)
│   └── check/
│       └── check.go                 # Standalone diagnostics (library)
├── Makefile                         # Build, install, test, release targets
├── .goreleaser.yml                  # GoReleaser config for cross-platform releases
├── .github/workflows/release.yml    # GitHub Actions CI/CD
├── packaging/
│   ├── homebrew/bwenv.rb            # Homebrew formula template
│   └── scoop/bwenv.json             # Scoop manifest template
├── LICENSE
└── README.md
```

---

## 🔧 Development

### Build

```bash
make build        # Build for current platform → dist/bwenv
make run          # Build and run
make run ARGS="status"  # Build and run with arguments
```

### Test

```bash
make test         # Run all Go tests
make lint         # Run go vet + staticcheck
make fmt          # Format all Go source files
```

### Release

```bash
# Local test build (no publish)
goreleaser release --snapshot --clean

# Full release (requires GITHUB_TOKEN)
goreleaser release --clean

# Or use the Makefile for a simple cross-compile
make release
```

### Adding a New Provider

1. Create a new file in `internal/provider/` (e.g. `doppler.go`)
2. Implement the `Provider` interface (including the `Lock()` method)
3. Call `Register(&YourProvider{})` in an `init()` function
4. That's it — the provider will automatically appear in the TUI picker and CLI flags

---

## 🤝 Supported Providers

| Provider | CLI Tool | Status | Notes |
|----------|----------|--------|-------|
| **Bitwarden** | `bw` | ✅ Ready | Reads custom fields from items in folders |
| **1Password** | `op` | ✅ Ready | Reads fields from items in vaults |

> Want another provider? [Open an issue](https://github.com/s1ks1/bwenv/issues) or submit a PR! The provider interface is designed to be easy to extend.

---

## 📋 Migration from v1 (Bash)

If you're upgrading from the original Bash-based bwenv:

1. **Uninstall the old version:**
   ```bash
   # If installed via the old install.sh or make:
   rm -f ~/.local/bin/bwenv
   rm -f ~/.config/direnv/lib/bitwarden_folders.sh

   # If installed via Homebrew:
   brew uninstall --cask bwenv
   ```

2. **Install the new version:**
   ```bash
   brew tap s1ks1/bwenv
   brew install --cask bwenv
   ```

3. **Re-initialize your projects:**
   ```bash
   cd your-project
   bwenv init    # New interactive TUI flow
   direnv allow
   ```

4. **Configure preferences (optional):**
   ```bash
   bwenv config  # Toggle emoji, direnv output, etc.
   ```

### What changed?

| | v1 (Bash) | v2 (Go) |
|---|---|---|
| Language | Bash + batch scripts | Go (single binary) |
| Providers | Bitwarden only | Bitwarden + 1Password (extensible) |
| Dependencies | `bw`, `jq`, `direnv` | `bw` or `op`, `direnv` (no `jq` needed!) |
| UI | Basic terminal prompts | Beautiful TUI with Bubble Tea + Lipgloss |
| Windows | `.bat` file with PowerShell fallbacks | Native `.exe` binary |
| Helper scripts | `bitwarden_folders.sh` + `bwenv` bash script | None — everything is in the single binary |
| Config | None | Persistent preferences via `bwenv config` |
| Session management | Manual | `bwenv login` to re-auth, `bwenv logout` to lock vaults |
| Status overview | None | `bwenv status` for quick state check |

---

## 📝 License

MIT License. See [LICENSE](LICENSE) for details.

---

## 🤝 Contributing

Pull requests are welcome! For major changes, please open an issue first to discuss what you'd like to change.

The codebase is intentionally well-commented to make it easy for contributors who may not be deeply familiar with Go, Bubble Tea, or Lipgloss.

---

<div align="center">
  <b>Made with ❤️ for developers who care about security and beautiful tools</b>
</div>