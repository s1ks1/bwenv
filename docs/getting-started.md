# Your first project

[Documentation home](../README.md) · [Installation](../INSTALL.md) · [Hook setup](activation.md)

This walkthrough uses harmless demo data. Complete [installation](../INSTALL.md)
first. The commands run inside a project directory; no `.env` file is needed.

## 1. Prepare your secret fields

Create a dedicated folder/vault named `bwenv-demo`, then add an item with these fields:

| Field name/label | Demo value | Purpose |
| --- | --- | --- |
| `BWENV_DEMO` | `hello-bwenv` | Confirms setup without real credentials |
| `DB_HOST` | `localhost` | Example application setting |
| `DB_PORT` | `5432` | Environment values are strings |

<details open>
<summary>Bitwarden: folder and custom fields</summary>

1. Install `bw` and run `bw login` once to establish your account.
2. In the Bitwarden app/web vault, create the folder `bwenv-demo`.
3. Add a Secure Note or Login item to that folder.
4. Add the table's names and values as **custom fields**.

bwenv reads named custom fields, including hidden fields. Standard Bitwarden
username/password fields are not exported automatically; add explicitly named
custom fields for values your application needs. Empty named custom fields can
produce empty environment values.

If you added items through another client after the CLI's last sync, sync an
unlocked CLI session before setup, or use `bwenv refresh` after login in the project.
When Bitwarden needs unlocking, interactive authentication attempts a best-effort
sync before unlock; a valid existing session is reused without that sync. Use
`bwenv refresh` when you need an explicit sync with reported failure. Automatic
exports never sync.

</details>

<details>
<summary>1Password: vault and labeled item fields</summary>

1. Install `op` v2 and configure your account following the
   [official setup guide](https://developer.1password.com/docs/cli/get-started/).
2. For interactive desktop use, enable the app's CLI integration. Keep the app
   available for the authentication prompt.
3. In the 1Password app, create or choose a vault named `bwenv-demo`.
4. Add an item with custom fields labeled as shown in the table.

bwenv reads labeled non-empty fields, including standard username/password fields
when labeled. It skips notes, OTP fields, unlabeled fields and empty values. Select
only the items you intend to expose. Explicit selection fails if a selected item
cannot be fetched; whole-vault loading is best-effort and can warn while loading
only the accessible items. For automation, select explicit IDs and verify required
variables. With no desktop integration, establish a
valid `OP_SESSION_*` session using the provider's instructions before using bwenv;
bwenv does not import the token-bearing stdout from `op signin`.

</details>

### Field naming rules

Use unique names matching `[A-Za-z_][A-Za-z0-9_]*`, ideally uppercase with
underscores. Avoid shell configuration names such as `PATH` and bwenv's internal
control names. Names retain their case; bwenv does not uppercase them.

| Input field | Exported name |
| --- | --- |
| `API_KEY` | `API_KEY` |
| `DB-HOST` | `DB_HOST` |
| `1TOKEN` | `_1TOKEN` |

Non-ASCII characters and punctuation become underscores. Different labels may
therefore collide, for example `DB-HOST` and `DB HOST`. Duplicate names overwrite
in emitted order; do not rely on provider ordering to choose a value. Use one
unique variable name across the entire selected set. Values are shell-quoted;
NUL bytes and reserved control names are rejected.

## 2. Choose a hook

For your first project, keep the native `shell` default. It needs no additional CLI.
If you prefer direnv or mise across new projects, run:

```sh
bwenv config
```

Select **Default Activation Hook**, press Enter to cycle, and **S** to save.
This is a user preference, not a prompt you must answer every time. An existing
project's `.bwenv.toml` remains authoritative. See [hook comparison](activation.md).

## 3. Initialize a project

```sh
mkdir -p ~/bwenv-demo
cd ~/bwenv-demo
bwenv init
```

Setup authenticates, asks for a folder/vault and item selection, previews variable
**names**, then writes configuration and shell integration. If only one provider
or item is available, setup selects it automatically.

| Picker key | Action |
| --- | --- |
| Up/Down or `k`/`j` | Move |
| `/` | Search/filter |
| Enter | Confirm |
| Space in item picker | Toggle one item |
| `a` in item picker | Toggle all visible items |
| Esc / `q` / Ctrl+C outside search | Cancel |

> [!IMPORTANT]
> An empty item selection means **all items in the folder/vault**, not zero
> items. Selecting every current item stores explicit IDs; future items are not
> automatically part of that selection. Inspect `.bwenv.toml` before committing it.

A missing preview or empty selection can produce a configuration that needs
follow-up. Do not treat the setup success box alone as proof that variables loaded.

## 4. Activate the shell integration once

Follow the **RC path printed by init**. For example:

```sh
source ~/.zshrc
```

Or open a new terminal. Bash normally uses `~/.bashrc`; macOS Bash may use
`~/.bash_profile`. Fish uses:

```fish
source ~/.config/fish/config.fish
```

If you chose direnv or mise, also enable its external hook and complete the trust
step in [Activation](activation.md). Native setup installs its hook itself.

This step makes `bwenv` a shell function around the executable. A child executable
cannot alter its parent terminal by itself; the wrapper applies successful shell
output for you. It is normal that authentication during `init` does not transfer
its temporary Bitwarden session into your shell.

## 5. Log in and verify

In the project:

```sh
bwenv login
bwenv doctor
bwenv status
```

Check the demo value without displaying it:

<details open>
<summary>Bash / Zsh</summary>

```sh
if [ "${BWENV_DEMO:-}" = hello-bwenv ]; then
  printf '%s\n' 'Demo variable loaded'
else
  printf '%s\n' 'Demo variable missing or incorrect'
fi
```

</details>

<details>
<summary>Fish</summary>

```fish
if test "$BWENV_DEMO" = hello-bwenv
    echo 'Demo variable loaded'
else
    echo 'Demo variable missing or incorrect'
end
```

</details>

Expected: `Demo variable loaded`. You can now start your application normally;
it inherits these variables from the terminal. Do not use full `env`, `printenv`
or shell tracing to verify real credentials in shared logs.

## 6. Work normally

| Situation | What to do |
| --- | --- |
| Return to the same project in an authenticated shell | Enter the directory; the hook loads variables |
| Stay inside project subdirectories | Keep working; native hooks reuse the loaded state |
| Open a new terminal and receive a login hint | Run `bwenv login` in the project |
| Change values in another Bitwarden client | Run `bwenv refresh` with an active session |
| Stop automatic loading for a project | Run `bwenv disallow` |
| Resume a disabled project | Run `bwenv allow` or `bwenv login` |
| Lock/sign out and clear this terminal | Run `bwenv lock` |

For native activation, leaving the project restores pre-existing values and
unsets variables that did not previously exist. Direnv/mise use their own
environment lifecycle. Provider login is distinct from project activation:
leaving a directory does not sign you out of the provider.

## 7. Clean up the demo

```sh
bwenv remove
cd ~
```

This removes bwenv configuration for the project and restores/clears managed
variables through the wrapper. It does not delete vault items or unrelated
project files. Delete the harmless demo folder/vault in your password manager
when you no longer need it. Shared shell integration remains installed.

Next: [configuration](configuration.md), [commands](commands.md), or
[automation](automation.md). If any step fails, use [troubleshooting](troubleshooting.md).
