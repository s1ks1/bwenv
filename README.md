<div align="center">
  <img src="./assets/Logo.svg" alt="bwenv Logo" width="120"/>
  <h1>🔐 bwenv</h1>
  <p><strong>Sync secrets from your password manager into your shell environment — beautifully.</strong></p>
  <p>
    <a href="#-installation"><img src="https://img.shields.io/badge/install-homebrew%20%7C%20scoop%20%7C%20go-blue" alt="Install"/></a>
    <a href="https://github.com/s1ks1/bwenv/releases"><img src="https://img.shields.io/github/v/release/s1ks1/bwenv?style=flat&color=green" alt="Release"/></a>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-purple" alt="License"/></a>
    <a href="https://github.com/s1ks1/bwenv/actions"><img src="https://img.shields.io/github/actions/workflow/status/s1ks1/bwenv/release.yml?label=build" alt="Build"/></a>
  </p>
</div>

---

## 🚀 Overview

**bwenv** is a cross-platform CLI tool that bridges your password manager and your shell environment using [direnv](https://direnv.net/). It lets you load secrets from **Bitwarden** or **1Password** directly into your project's environment variables — no manual copy-pasting, no secrets in `.env` files committed to git.

Built with [Go](https://go.dev/), [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lipgloss](https://github.com/charmbracelet/lipgloss) for a fast, beautiful, and truly cross-platform experience.

### Why the rewrite?

The original bwenv was built with Bash scripts, which worked — but had constant cross-platform issues between macOS, Linux, and Windows. This Go rewrite solves that by compiling to a **single static binary** for every platform, with zero runtime dependencies (beyond your password manager's CLI).

---

## ✨ Features

- **🔑 Multi-provider support** — Works with Bitwarden (`bw` CLI) and 1Password (`op` CLI)
- **🎨 Beautiful TUI** — Interactive provider and folder selection with arrow keys, search, and filtering
- **📁 Automatic `.envrc` generation** — Creates direnv-compatible files that auto-load your secrets
- **🖥️ True cross-platform** — Single binary for Linux, macOS, and Windows (amd64 + arm64)
- **🔍 Smart diagnostics** — `bwenv test` checks every dependency and shows a styled status report
- **⚡ Zero runtime dependencies** — Just the Go binary + your password manager CLI + direnv
- **📦 Easy installation** — Homebrew, Scoop, `go install`, or direct download

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

### Homebrew (macOS / Linux)

```bash
brew tap s1ks1/bwenv
brew install bwenv
```

### Scoop (Windows)

```powershell
scoop bucket add bwenv https://github.com/s1ks1/scoop-bwenv
scoop install bwenv
```

### Go Install

```bash
go install github.com/s1ks1/bwenv@latest
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
bwenv test
```

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
direnv allow
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

### 3. Check Dependencies

```bash
bwenv test
```

Prints a comprehensive diagnostic report showing:
- System info (OS, architecture, shell)
- Installed CLI tools and their versions
- Provider authentication status
- Direnv hook configuration
- Environment variables

### 4. Remove Secrets

```bash
bwenv remove
```

Deletes the `.envrc` file from the current directory.

### 5. Version

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
│   │   └── init_flow.go             # Orchestrates the full init TUI flow
│   ├── envrc/
│   │   └── envrc.go                 # .envrc generation and secret export
│   └── check/
│       └── check.go                 # Diagnostics for "bwenv test"
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
make run ARGS="test"  # Build and run with arguments
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
2. Implement the `Provider` interface
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
   brew uninstall bwenv
   ```

2. **Install the new version:**
   ```bash
   brew tap s1ks1/bwenv
   brew install bwenv
   ```

3. **Re-initialize your projects:**
   ```bash
   cd your-project
   bwenv init    # New interactive TUI flow
   direnv allow
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