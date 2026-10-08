# Install bwenv v3

[Documentation home](README.md) · [Next: your first project](docs/getting-started.md)

> [!NOTE]
> This guide targets **bwenv 3.x**. Examples that pin an exact release use
> **v3.0.0**. For a newer v3 patch release, substitute its tag consistently in
> download, installation and source-build commands.

## Requirements

| Requirement | Details |
| --- | --- |
| bwenv | One binary on your shell's `PATH` |
| Password manager CLI | Bitwarden `bw`, or 1Password `op` v2 configured for your account |
| Interactive shell | Bash, Zsh or Fish for automatic activation and the login wrapper |
| Optional hook | `direnv` or `mise` only when you choose that backend |
| Building from source | Go 1.25+; Make optional |

Release packages target macOS/Linux amd64 and arm64, and Windows amd64.
Windows has a CLI binary, but native PowerShell environment activation is not
implemented. For a supported workflow use Bash in Git Bash, or install the Linux
build and provider CLI inside WSL. Do not mix Windows and WSL installations or sessions.

Install the provider CLI using its official guide:

- [Bitwarden CLI](https://bitwarden.com/help/cli/).
- [1Password CLI](https://developer.1password.com/docs/cli/get-started/).

Verify the executable in the **same shell** that will run bwenv:

```sh
bw --version  # Bitwarden users
op --version  # 1Password users
```

You only need one of these. If you choose an optional backend, verify `direnv version`
or `mise --version` too.

## Choose an installation method

Choose one method for your platform. After installation, run `bwenv --version`
and confirm a **v3.x** version. Use the same method for future upgrades to avoid
multiple binaries taking precedence on PATH.

<details>
<summary>Homebrew on macOS</summary>

```sh
brew tap s1ks1/bwenv
brew install --cask bwenv
```

Upgrade with `brew upgrade --cask bwenv`.

</details>

<details>
<summary>Scoop on Windows</summary>

```powershell
scoop bucket add bwenv https://github.com/s1ks1/scoop-bwenv
scoop install bwenv
```

Upgrade with `scoop update bwenv`. Use a supported shell for activation.

</details>

<details>
<summary>Go module installation</summary>

Install the initial v3 release:

```sh
go install github.com/s1ks1/bwenv/v3@v3.0.0
```

For the latest release in the v3 module, use
`go install github.com/s1ks1/bwenv/v3@latest`. Go installation requires Go 1.25+.

</details>

<details>
<summary>Install scripts</summary>

Download and inspect the script before executing it:

```sh
curl -fsSL https://raw.githubusercontent.com/s1ks1/bwenv/v3.0.0/install.sh -o /tmp/bwenv-install.sh
less /tmp/bwenv-install.sh
BWENV_VERSION=v3.0.0 sh /tmp/bwenv-install.sh
```

The POSIX installer accepts `BWENV_VERSION` (a published tag) and `BWENV_DIR`:

```sh
BWENV_VERSION=v3.0.0 BWENV_DIR="$HOME/.local/bin" sh /tmp/bwenv-install.sh
```

Keep the script URL and `BWENV_VERSION` on the same release tag when pinning a version.

PowerShell:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/s1ks1/bwenv/v3.0.0/install.ps1 -OutFile bwenv-install.ps1
Get-Content .\bwenv-install.ps1
.\bwenv-install.ps1 -Version v3.0.0
```

Use `-Version v3.0.0 -InstallDir <directory>` to select a release and destination.
The installers require an exact, unique SHA256 checksum before installation.
Do not bypass a verification failure.

</details>

<details>
<summary>Manual archives and Linux DEB/RPM packages</summary>

Open [Releases](https://github.com/s1ks1/bwenv/releases) and choose a **v3** release,
OS and architecture. Download its `checksums.txt` and the matching archive/package.
Check the filename and digest **before** extracting or installing:

```sh
sha256sum YOUR_DOWNLOADED_FILE     # Linux
shasum -a 256 YOUR_DOWNLOADED_FILE # macOS
```

```powershell
Get-FileHash .\YOUR_DOWNLOADED_FILE -Algorithm SHA256
```

Compare the complete digest with the exact filename in `checksums.txt`.
For Linux packages, install the verified file with `sudo dpkg -i ./FILE.deb`
or `sudo rpm -i ./FILE.rpm`. Names shown here are placeholders.
For archives, extract and place `bwenv` / `bwenv.exe` on PATH.

</details>

## Build v3 from source

Build the tagged release to use the same source version as the packaged binary:

```sh
git clone --branch v3.0.0 --depth 1 https://github.com/s1ks1/bwenv.git
cd bwenv
go mod download
make build
make install
```

`make install` builds and copies the binary to `~/.local/bin/bwenv`.
Use a clean release-tag checkout for release version metadata. Builds from
modified or untagged source are development builds, not the official release binary.

<details>
<summary>Build without Make, including Windows</summary>

Run from the clean release-tag checkout. Match the injected version to that tag:

```sh
go build -ldflags "-X main.Version=v3.0.0" -o bwenv ./cmd/bwenv
```

In PowerShell:

```powershell
go build -ldflags "-X main.Version=v3.0.0" -o bwenv.exe ./cmd/bwenv
```

Move the binary to a directory on your `PATH`. Do not commit build output.
For automatic loading on Windows, continue in a supported shell.

</details>

## Make the binary available on PATH

If `command -v bwenv` finds nothing, add the installation directory to your shell
RC file once. Keep any existing PATH configuration.

| Shell | RC file normally used | Line to add |
| --- | --- | --- |
| Zsh | `~/.zshrc` | `export PATH="$HOME/.local/bin:$PATH"` |
| Bash | `~/.bashrc` | `export PATH="$HOME/.local/bin:$PATH"` |
| Fish | `~/.config/fish/config.fish` | `fish_add_path ~/.local/bin` |

On macOS Bash, bwenv uses an existing `~/.bash_profile` when present. Login shells
may need that file to source `~/.bashrc`. Follow the actual RC path printed by init.
For `go install`, use the Go bin directory (`go env GOBIN`, or `$(go env GOPATH)/bin`
when GOBIN is empty) instead of `~/.local/bin`.

Open a new terminal after changing PATH:

```sh
command -v bwenv
bwenv --version
```

## Finish setup

1. Prepare your password manager account and a test folder/vault.
2. Run `bwenv init` inside your project.
3. Source the RC file printed by setup, or open a new terminal.
4. If using direnv or mise, enable its [one-time shell hook](docs/activation.md).
5. In the project, run `bwenv login`, then `bwenv doctor`.

Continue with [Getting started](docs/getting-started.md). If installation or shell
setup fails, use [Troubleshooting](docs/troubleshooting.md).

## Download or retain v2

The [v2.4.1 release](https://github.com/s1ks1/bwenv/releases/tag/v2.4.1) and its
platform archives remain available. Homebrew, Scoop and the default install
scripts follow the current V3 release. To stay on V2, download its archive and
verify it against that release's `checksums.txt`, or pin the Go module:

```sh
go install github.com/s1ks1/bwenv/v2@v2.4.1
```

The `v2` branch retains the released V2 source for possible patches. Future V2
patches will keep their own tags and downloads without replacing V3 as latest.
Before switching a configured V3 project back to V2, preserve its metadata and
use a compatible direnv setup: V2 does not support native or mise activation.

## Uninstall

First run `bwenv remove` in each project you no longer want managed, while the
shell wrapper is still installed. Lock/sign out as appropriate with `bwenv lock`.
Then uninstall through the tool that installed the binary (`brew uninstall --cask bwenv`,
`scoop uninstall bwenv`, or `make uninstall` for the default source installation).
For a manual installation, remove that specific binary.

Review your RC file and remove bwenv's generated hook, wrapper and login-notice
blocks if no projects use them. `bwenv remove` leaves shared shell integration
installed. Keep direnv/mise hooks if other projects need them. User preferences
remain in `~/.config/bwenv` (or `$XDG_CONFIG_HOME/bwenv`); remove them only if you
want to discard those preferences too.
