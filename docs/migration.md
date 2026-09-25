# Project metadata and migration

## `.bwenv.toml` in v2.4

When `bwenv init` has a stable folder ID, it writes a versioned `.bwenv.toml` alongside `.envrc`.
This file is project configuration and can be committed with the project:

```toml
version = 1
provider = "bitwarden"

[project]
folder_id = "abc123"
folder_name = "Production"
items = ["item-id-a", "item-id-b"]

[activation]
mode = "direnv"
```

The file stores provider references and activation settings, not secret values or session tokens.
Bitwarden's `BW_SESSION`, when needed, remains in the permission-restricted `.envrc` in this version.

Generated `.envrc` files load this metadata with `bwenv export --project .`. The direct `--provider`, `--folder`, `--folder-id`, and `--items` flags remain available for scripts and CI.

Existing projects continue to work without `.bwenv.toml`: bwenv reads their generated `.envrc`
settings. If a `.bwenv.toml` file exists but is invalid or uses an unsupported version, bwenv
reports the error instead of silently using potentially stale `.envrc` metadata.

Projects without a stable folder ID continue using the legacy `.envrc` settings and do not receive a
canonical project file. No automatic migration command is included in v2.4; the migration workflow
is tracked separately.

## Upgrading from v1

The original v1 project used Makefile, Bash, and PowerShell scripts. The current v2 application is
a Go binary, introduced to provide consistent cross-platform behavior and native support across
operating systems. Install the current release, then run `bwenv init` in each project to generate
its current activation files. See the [v1 migration steps in the README](../README.md#migration-from-the-original-version).
