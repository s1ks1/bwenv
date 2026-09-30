<div align="center">

  <img src="./assets/Logo.svg" alt="bwenv Logo" width="120"/>
  <h1>🔐 bwenv</h1>

  <p><em>Sync secrets from your password manager into your shell environment — beautifully.</em></p>

  <p>
    <a href="#-phase-1-install"><img src="https://img.shields.io/badge/install-homebrew%20%7C%20scoop%20%7C%20apt%20%7C%20go-blue" alt="Install"/></a>
    <a href="https://github.com/s1ks1/bwenv/releases"><img src="https://img.shields.io/github/v/release/s1ks1/bwenv?style=flat&color=green" alt="Release"/></a>
    <a href="https://github.com/s1ks1/bwenv/actions"><img src="https://img.shields.io/github/actions/workflow/status/s1ks1/bwenv/release.yml?label=build" alt="Build"/></a>
    <br/>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-purple" alt="License"/></a>
    <a href="https://goreportcard.com/report/github.com/s1ks1/bwenv"><img src="https://goreportcard.com/badge/github.com/s1ks1/bwenv" alt="Go Report Card"/></a>
  </p>

  <p><strong>📖 Read in five phases</strong></p>

  <table align="center">
    <tr>
      <th align="center">📦<br>Install</th>
      <th align="center">⚡<br>Quick start</th>
      <th align="center">🔄<br>Everyday use</th>
      <th align="center">🧩<br>Advanced</th>
      <th align="center">🔧<br>Development</th>
    </tr>
    <tr>
      <td align="center"><b><a href="#-phase-1-install">Phase 1 →</a></b></td>
      <td align="center"><b><a href="#-phase-2-quick-start">Phase 2 →</a></b></td>
      <td align="center"><b><a href="#-phase-3-everyday-use">Phase 3 →</a></b></td>
      <td align="center"><b><a href="#-phase-4-advanced">Phase 4 →</a></b></td>
      <td align="center"><b><a href="#-phase-5-development">Phase 5 →</a></b></td>
    </tr>
  </table>

</div>

---

## 🎯 What is bwenv?

**bwenv** bridges your password manager and your shell using a native hook, [direnv](https://direnv.net/), or [mise](https://mise.jdx.dev/). It loads secrets from **Bitwarden** or **1Password** into your project's environment variables — no copy-pasting, no `.env` files committed to git.

Built with [Go](https://go.dev/), [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lipgloss](https://github.com/charmbracelet/lipgloss). One static binary per platform, the native hook needs only your password manager CLI.

### Why bwenv?

`.env` files get committed by accident, tokens expire and break your workflow, and switching projects means manual copy-pasting. After login in a shell, bwenv fetches secrets live from your vault as you enter configured directories.

The original bwenv was a collection of Makefile, Bash, and PowerShell scripts. The Go rewrite keeps behavior consistent across macOS, Linux, and Windows with a single static binary.

### Why bwenv for AI workflows?

AI coding assistants read your project files and environment to understand context. bwenv keeps secrets out of that context:

- **Nothing to leak** — secret values stay in your vault and are fetched live; keys and passwords are never written to disk.
- **AI sees no secret values** — `.envrc` holds only a reference to `bwenv export`; provider session tokens stay in the current shell, not project files.
- **Available in the shell** — secrets load as environment variables that AI tools inherit from your terminal.
- **No `.env` to commit** — keep generated `.envrc` out of git; it contains only the activation command and is written with mode `0600`.
- **Share context safely** — `.bwenv.toml` contains provider references, not credentials. Review custom `.envrc` code before sharing it.

### Features

- **🎯 Pinpoint selection** — load specific items via TUI multi-select or the `--items` flag
- **🔑 Multi-provider** — Bitwarden (`bw`) and 1Password (`op`)
- **🎨 Interactive TUI** — folder browsing with search and filtering
- **📁 Automatic `.envrc` generation** — direnv-compatible files that load secrets on `cd`
- **🖥️ True cross-platform** — Linux, macOS, Windows (amd64 + arm64)
- **🔍 Smart diagnostics** — `bwenv doctor` checks setup safely; `bwenv status` shows full context
- **⚙️ Configurable UI** — toggle emoji, direnv output, export summaries via `bwenv config`
- **🔑 Quick re-auth** — `bwenv login` re-authenticates and updates `.envrc` in one step
- **🔒 Secure logout** — lock vaults and terminate sessions with `bwenv logout`

---

## 📦 Phase 1: Install

### Prerequisites

| Tool | Required? | Description |
|------|-----------|-------------|
| [direnv](https://direnv.net/) | Optional | Loads and unloads environment variables from `.envrc` files |
| [Bitwarden CLI](https://bitwarden.com/help/cli/) | One of these | Access your Bitwarden vault (`bw`) |
| [1Password CLI](https://developer.1password.com/docs/cli/) | One of these | Access your 1Password vaults (`op`) |

You need **at least one** password manager CLI. bwenv detects what's installed and lets you choose.

**Log into your password manager CLI before running bwenv:**

- **Bitwarden:** run `bw login` once. `bwenv init` then prompts for your master password to unlock.
- **1Password:** run `op signin`, or rely on desktop app biometrics (op v2).

If the CLI is not logged in, bwenv fails at the authentication step. See [INSTALL.md](INSTALL.md) for detailed setup instructions.

### Choose your activation hook once

In v3, new projects default to **shell**, a native Bash/Zsh/Fish hook with no direnv or mise dependency. Native hooks and mise remain experimental; direnv is the established option.

Run `bwenv config`, select **Default Activation Hook**, press Enter to cycle through `shell`, `direnv`, and `mise`, then **S** to save. The choice applies to new projects. Existing projects keep their `.bwenv.toml` mode; `bwenv init --activation mise` overrides one project's choice.

`bwenv init` shows which hook it uses and the exact setup commands. For native activation, source the displayed RC file (or open a new terminal), then run `bwenv login`. Manual Bash/Zsh setup:

```bash
eval "$(bwenv hook zsh)" # use bash for Bash
```

Mise sources the generated script with `tools = true`, so Node-based password managers use the active runtime without recursing through mise shims. Automatic mise loads stay silent; the shell shows a login hint once per project entry when the vault is locked. Tool activation still completes; run `bwenv login` to unlock it or `bwenv export --project .` to see an error.

For Fish: `bwenv hook fish | source`. For direnv, enable `eval "$(direnv hook zsh)"`; for mise, enable `eval "$(mise activate zsh)"` and run `mise trust` once per project. Add the relevant command to your RC file for future sessions. PowerShell users should select direnv; the native hook supports Bash/Zsh/Fish.

The native hook restores exported variables when leaving or switching projects, including previous values. It stores this state only in the shell environment, never on disk. Nested directories and repeated sourcing do not fetch secrets again. If activation fails, one concise warning is shown. Run `bwenv login` directly: the wrapper evaluates it automatically. Re-entering a project in the same shell reuses the session. A new shell or expired session requires login; session tokens are not saved on disk.

### Install bwenv

**Homebrew (macOS):**

```bash
brew tap s1ks1/bwenv
brew install --cask bwenv
```

**Scoop (Windows):**

```powershell
scoop bucket add bwenv https://github.com/s1ks1/scoop-bwenv
scoop install bwenv
```

**Linux (DEB — Debian / Ubuntu):**

```bash
curl -LO https://github.com/s1ks1/bwenv/releases/latest/download/bwenv_VERSION_amd64.deb
sudo dpkg -i bwenv_*_amd64.deb
```

**Linux (RPM — Fedora / RHEL / openSUSE):**

```bash
curl -LO https://github.com/s1ks1/bwenv/releases/latest/download/bwenv_VERSION_amd64.rpm
sudo rpm -i bwenv_*_amd64.rpm
```

**Go:**

```bash
go install github.com/s1ks1/bwenv/v3@latest
```

**Install script (macOS / Linux):**

```bash
curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/main/install.sh | sh
```

**Install script (Windows PowerShell):**

```powershell
irm https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1 | iex
```

**From source:**

```bash
git clone https://github.com/s1ks1/bwenv.git
cd bwenv
make build
make install
```

**Direct download:** grab the binary for your platform from the [Releases](https://github.com/s1ks1/bwenv/releases) page, extract it, and put `bwenv` on your `PATH`.

### Verify the install

```bash
bwenv status
```

> Full platform-by-platform instructions, including test workflows for Bitwarden and 1Password, are in [INSTALL.md](INSTALL.md).

---

## ⚡ Phase 2: Quick start

Set up a project in one interactive run:

```bash
bwenv init
```

The TUI walks you through five steps:

1. **Select a provider** — Bitwarden or 1Password (only installed CLIs appear)
2. **Authenticate** — unlock your vault (master password, biometrics, and so on)
3. **Pick a folder** — browse and search for the folder that holds your secrets
4. **Pick items** — load the whole folder or select specific items
5. **Configure activation** — bwenv writes `.bwenv.toml` and installs your selected hook

Authenticate and load secrets into this shell:

```bash
bwenv login
```

The shell integration evaluates the login output. In shells without the integration, use `eval "$(bwenv login)"` in Bash/Zsh or `eval (bwenv login)` in Fish. Once logged in, your selected hook loads project secrets on entry and restores the environment on exit. Run login again in each new shell session.

---

## 🔄 Phase 3: Everyday use

### Session expired? Re-authenticate

```bash
bwenv login
```

`bwenv login` reads the provider from `.bwenv.toml`, authenticates, and loads secrets into the current shell. Run it once in each new shell session. The shell integration installed by `bwenv init` evaluates its output automatically; without that integration, use `eval "$(bwenv login)"` in Bash/Zsh or `eval (bwenv login)` in Fish.

> **Alias:** `bwenv auth` works too.

### Lock vaults and log out

```bash
bwenv logout
```

- **Bitwarden** — runs `bw lock`
- **1Password** — runs `op signout`
- Shows any lingering session environment variables and how to clear them

Use it when you're done working with secrets or stepping away from your machine.

### Refresh provider data

```bash
bwenv refresh
```

Refresh requires an active provider session and a working direnv hook. Bitwarden runs an explicit `bw sync` before direnv reloads the project environment. 1Password has no separate local sync step; direnv re-runs the export. `bwenv export` itself never syncs, so ordinary directory changes do not trigger a vault sync.

### Check your setup

```bash
bwenv doctor
bwenv status
```

`bwenv doctor` runs actionable setup checks and returns a non-zero exit code when a required check fails. Its output is safe to share in an issue report: no secret values, session tokens, provider payloads, or project paths. It also flags older `.envrc` files without a FolderID and recommends regenerating them with `bwenv init`.

`bwenv status` shows a full overview:

- Current directory and `.envrc` info (provider, folder)
- direnv installation and hook status
- Provider availability and active sessions
- Relevant environment variables (masked)
- Current config preferences

### Tune the UI

```bash
bwenv config
```

| Setting | Default | Description |
|---------|---------|-------------|
| **Default Activation Hook** | shell | Hook for new projects; direnv and mise are optional |
| **Show Emoji** | ON | Emoji icons in output (off for text-only output) |
| **Show Direnv Output** | OFF | direnv's own loading/unloading messages |
| **Show Export Summary** | ON | Compact load summary for shell, direnv, and mise |

Settings persist to `~/.config/bwenv/config.json`.

### Remove secrets from a project

```bash
bwenv remove
```

Deletes the `.envrc` file from the current directory.

### Command cheat sheet

| Command | What it does |
|---------|--------------|
| `bwenv init` | Interactive project setup |
| `bwenv login` (`auth`) | Authenticate and load secrets into the current shell |
| `bwenv logout` | Lock vaults and terminate sessions |
| `bwenv refresh` | Sync provider data and reload the environment |
| `bwenv status` | Full state overview |
| `bwenv doctor` | Shareable setup diagnostics |
| `bwenv config` | Edit UI preferences |
| `bwenv export` | Print secrets as `export KEY=VALUE` lines |
| `bwenv migrate` | Move a legacy project to `.bwenv.toml` |
| `bwenv remove` | Delete `.envrc` |
| `bwenv version` | Print version |

---

## 🧩 Phase 4: Advanced

### Export secrets without the TUI

For CI/CD pipelines, scripts, or advanced usage, export directly:

```bash
# Output "export KEY=VALUE" lines to stdout
bwenv export --provider bitwarden --folder "MySecrets"

# Use the persisted folder ID for the fast path (generated .envrc files do this automatically)
bwenv export --provider bitwarden --folder-id "provider-folder-id" --folder "MySecrets"

# Export only specific items from a folder
bwenv export --provider bitwarden --folder "MySecrets" --items "item-id-1,item-id-2"

# Use with eval to set variables in the current shell
eval "$(bwenv export --provider bitwarden --folder "MySecrets")"

# Works with 1Password too
eval "$(bwenv export --provider 1password --folder "Production")"

# Load canonical settings from a bwenv project
bwenv export --project .
```

### Project metadata: `.bwenv.toml`

New projects with a stable folder ID store versioned, secret-free references in `.bwenv.toml`: provider, folder, item references, and activation mode. It contains no secret values and can be committed. Generated `.envrc` files load it with `bwenv export --project .`.

Existing `.envrc` projects and direct export flags keep working. To move an older project to `.bwenv.toml`:

```bash
bwenv migrate --dry-run   # preview the changes
bwenv migrate             # apply them
```

Migration keeps the original file as `.envrc.bwenv.bak` until you verify the project. See [docs/migration.md](docs/migration.md) for the format and compatibility details.

### How bwenv works

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

1. **`bwenv init`** walks you through provider, folder, and item selection.
2. It generates an `.envrc` with a single `eval` call to `bwenv export`.
3. When direnv loads `.envrc`, it runs `bwenv export`, which fetches fresh secrets from your vault.
4. Each secret's custom fields (Bitwarden) or item fields (1Password) become environment variables.

**No secret values or session tokens are stored on disk.** Every direnv load fetches secrets live using the session in the current shell.

### Supported providers

| Provider | CLI Tool | Status | Notes |
|----------|----------|--------|-------|
| **Bitwarden** | `bw` | ✅ Ready | Reads custom fields from items in folders |
| **1Password** | `op` | ✅ Ready | Reads fields from items in vaults |

> Want another provider? [Open an issue](https://github.com/s1ks1/bwenv/issues) or submit a PR. The provider interface is designed to be easy to extend.

### Upgrade from the script-based v1

If you still run the original Makefile, Bash, and PowerShell version:

1. **Uninstall the old version:**
   ```bash
   # Installed via the old install.sh or make:
   rm -f ~/.local/bin/bwenv
   rm -f ~/.config/direnv/lib/bitwarden_folders.sh

   # Installed via Homebrew:
   brew uninstall --cask bwenv
   ```

2. **Install the Go release:**
   ```bash
   brew tap s1ks1/bwenv
   brew install --cask bwenv
   ```

3. **Re-initialize your projects:**
   ```bash
   cd your-project
   bwenv init    # new interactive TUI flow
   direnv allow
   ```

4. **Configure preferences (optional):** `bwenv config`

| | v1 (scripts) | v2 (Go) |
|---|---|---|
| Implementation | Makefile, Bash, and PowerShell scripts | Go (single binary) |
| Providers | Bitwarden only | Bitwarden + 1Password (extensible) |
| Dependencies | `bw`, `jq`, `direnv` | `bw` or `op`, `direnv` (no `jq` needed!) |
| UI | Basic terminal prompts | TUI with Bubble Tea + Lipgloss |
| Windows | `.bat` file with PowerShell fallbacks | Native `.exe` binary |
| Helper scripts | `bitwarden_folders.sh` + `bwenv` bash script | None — everything is in the single binary |
| Config | None | Persistent preferences via `bwenv config` |
| Session management | Manual | `bwenv login` to re-auth, `bwenv logout` to lock vaults |
| Status overview | None | `bwenv status` for a quick state check |

---

## 🔧 Phase 5: Development

### Build and test

```bash
make build                # Build for the current platform → dist/bwenv
make run                  # Build and run
make run ARGS="status"    # Build and run with arguments
make test                 # Run all Go tests
make lint                 # Run go vet + staticcheck
make fmt                  # Format all Go source files
make tidy                 # Clean up go.mod/go.sum
```

Run `bwenv benchmark` inside a configured project to measure provider calls without displaying secret values. See [docs/performance.md](docs/performance.md) and the [changelog](CHANGELOG.md) for the current development milestone.

### Project structure

```
bwenv/
├── main.go                  # Entry point and CLI routing
├── internal/
│   ├── provider/            # Provider interface + Bitwarden and 1Password
│   ├── ui/                  # TUI flows and styles (Bubble Tea, Lipgloss)
│   ├── envrc/               # .envrc and .bwenv.toml generation, export, migrate
│   ├── config/              # Persistent preferences (~/.config/bwenv/)
│   ├── process/             # Child-process runner
│   ├── benchmark/           # bwenv benchmark implementation
│   └── diagnostics/         # doctor and status checks
├── install.sh               # macOS/Linux quick install script
├── install.ps1              # Windows quick install script
├── packaging/               # Homebrew, Scoop, Windows shim
├── Makefile                 # Build, install, test, release targets
├── .goreleaser.yml          # GoReleaser config for cross-platform releases
├── .github/workflows/       # GitHub Actions workflows
├── docs/                    # Migration guide, performance notes
├── INSTALL.md               # Detailed install and testing guide
├── CHANGELOG.md
└── LICENSE
```

### Git workflow

`main` is the stable branch and release source. Work on feature branches, open a pull request, and merge when checks pass. A fresh clone builds immediately:

```bash
git clone https://github.com/s1ks1/bwenv.git
cd bwenv
git switch main
make build
```

For local pulls, prefer merge-based updates:

```bash
git config pull.rebase false
git pull origin main
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

### Add a new provider

1. Create a file in `internal/provider/` (for example `doppler.go`).
2. Implement the `Provider` interface, including the `Lock()` method.
3. Call `Register(&YourProvider{})` in an `init()` function.
4. The provider appears in the TUI picker and CLI flags automatically.

---

## 📝 License

MIT License. See [LICENSE](LICENSE) for details.

---

## 🤝 Contributing

Pull requests are welcome! For major changes, open an issue first to discuss what you'd like to change.

The codebase is intentionally well-commented to make it easy for contributors who may not be deeply familiar with Go, Bubble Tea, or Lipgloss.

---

<div align="center">
  <b>Made with ❤️ for developers who care about security and beautiful tools</b>
</div>
