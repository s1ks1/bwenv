# Configuration

[Documentation home](../README.md) · [Activation](activation.md) · [Commands](commands.md)

bwenv has two configuration levels: user preferences select defaults and presentation;
project metadata selects the provider source and activation for one project.

## User preferences

```sh
bwenv config
```

Use Up/Down to select a setting. Enter changes the hook or toggles a boolean;
**S** saves. Follow the picker footer for cancel controls. Preferences are stored
in `$XDG_CONFIG_HOME/bwenv/config.json`, or `~/.config/bwenv/config.json` when that
variable is unset.

| JSON key | Default | Meaning |
| --- | --- | --- |
| `activation_mode` | `"shell"` | Backend for newly initialized projects |
| `show_emoji` | `true` | Use emoji in messages; disable for text-only terminals |
| `show_direnv_output` | `false` | Show direnv's routine loading/unloading messages |
| `show_export_summary` | `true` | Show compact summaries for explicit loads and changed values |

Equivalent full preference file:

```json
{
  "activation_mode": "shell",
  "show_emoji": true,
  "show_direnv_output": false,
  "show_export_summary": true
}
```

Mise's automatic source stays quiet regardless of the summary preference. Direnv
silencing affects the shell's direnv messages, including other direnv projects;
review the RC changes produced by setup when that distinction matters.

## Project metadata

`bwenv init` generates `.bwenv.toml`. Example with placeholder IDs:

```toml
version = 1
provider = "bitwarden"

[project]
folder_id = "REPLACE_WITH_FOLDER_ID"
folder_name = "bwenv-demo"
items = ["REPLACE_WITH_ITEM_ID"]

[activation]
mode = "shell"
# disabled = true
```

Do not paste placeholder IDs into a working project. Use init to obtain actual IDs.
`version = 1` is the **metadata schema**, not the bwenv application version.

| Field | Meaning |
| --- | --- |
| `provider` | `bitwarden` or `1password` |
| `project.folder_name` | Folder name (Bitwarden) or vault name (1Password); required |
| `project.folder_id` | Stable provider ID; avoids a folder-name lookup |
| `project.items` | Explicit item IDs; omit or use `[]` to load all items |
| `activation.mode` | `shell`, `direnv` or `mise` |
| `activation.disabled` | `true` disables native/mise automatic loading; omitted means enabled |

Keep IDs and names aligned if moving a selection. With a stored ID, renaming the
folder display name is not a way to select a different provider source. Re-run
init to browse and change the selection safely. Unknown fields and unsupported
schema versions are rejected rather than silently ignored.

### Precedence

1. `bwenv init --activation MODE` overrides the existing mode or user preference
   for that setup.
2. Without an override, an existing `.bwenv.toml` supplies the mode.
3. For a new project, init uses the user preference `activation_mode`; its default
   is `shell`.
4. Legacy generated `.envrc` is used only when canonical metadata is absent.
   Invalid `.bwenv.toml` is an error, not a reason to fall back to old metadata.

Direct exports can use `--project PATH`, or explicit provider/folder/item flags.
Those two forms cannot be combined. `--project .` reads the specified directory's
file; it does not search its parents. Normal project commands use the nearest
project root, so they work from nested directories.

## What to put in Git

| File | Contains | Guidance |
| --- | --- | --- |
| `.bwenv.toml` | Provider references, selection and enable/disable setting | May be committed after reviewing IDs/names and desired team state |
| `.envrc` | Direnv activation command, possibly your custom shell code | Keep local by default; review all code before sharing/trusting |
| `.bwenv.mise.sh` | Generated activation code, no decrypted exports | Regenerate locally or commit only after code review |
| `mise.toml` | Tool settings and activation reference | Review before sharing; teammates still need local trust/authentication |
| `.bwenv_vars` | Variable names for cleanup | Keep local; no values, but names can disclose project context |
| `.envrc.bwenv.bak` | Legacy migration backup | Keep local; review for old sensitive content |

Suggested project `.gitignore` entries:

```gitignore
.bwenv_vars
.envrc
.envrc.bwenv.bak
.bwenv.mise.sh
```

If your team deliberately versions reviewed activation code, adapt these entries.
A committed reference is not a credential, but folder names, IDs and variable
names can still reveal internal context. Never put tokens or decrypted exports
in `.bwenv.toml`, RC files, `.envrc`, mise scripts or Git.

> [!NOTE]
> `bwenv disallow` changes `.bwenv.toml` for native/mise projects. If that file is
> tracked, inspect `git diff` before committing: disabling it may affect teammates.
> For session-only cleanup, use `bwenv lock` instead.
