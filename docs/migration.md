# Migration to v3

[Documentation home](../README.md) · [Installation](../INSTALL.md) · [Configuration](configuration.md)

v3 keeps supported generated direnv projects working and adds native/mise choices.
Upgrading the binary does not automatically switch existing projects to the
native default. Preserve each project's provider source and item selection.

## Existing `.bwenv.toml` projects

1. Install bwenv v3 using the [installation guide](../INSTALL.md), then confirm
   `bwenv --version` reports a v3.x release.
2. Review your canonical metadata and existing activation artifacts.
3. Run `bwenv init` to regenerate recognized integration using the existing mode,
   selecting the intended folder/items again.
4. Source the RC path printed by setup or open a new terminal.
5. Complete external hook/trust setup if needed, then run `bwenv login` and `bwenv doctor`.

The metadata schema remains `version = 1`. It is separate from the application
major version. To change a project's backend, follow [switching hooks](activation.md#switching-an-existing-project).
Changing the global default applies to new projects only.

## Legacy generated `.envrc` without canonical metadata

From the legacy project root, preview first:

```sh
bwenv migrate --dry-run
```

Then apply:

```sh
bwenv migrate
```

Migration creates `.bwenv.toml` with the existing provider/folder/item references
and `activation.mode = "direnv"`. It replaces the generated retrieval command
with `bwenv export --project .` and saves `.envrc.bwenv.bak` with restrictive POSIX
permissions. It does not switch to the native backend.

Recognized literal `BW_SESSION` assignments are removed from **both** the new
activation file and the generated backup; the backup contains a removal comment.
Do not assume other old backup files have been sanitized. Review existing files
and Git history, and rotate credentials that were exposed.

Migration refuses custom shell commands, ambiguous session assignments and
unsupported input instead of evaluating code or discarding behavior. A dry run
writes nothing. If a previous backup already exists, preserve/review it first;
migration does not silently overwrite it.

After migration:

```sh
bwenv login
bwenv doctor
```

Verify one known test field using the [presence checks](getting-started.md#5-log-in-and-verify).
Keep the reviewed backup until you have verified the project, then remove it
intentionally. Ignore it in Git. Migrated metadata can lack a stable folder ID;
re-run init to select the actual folder and save the fast-path ID.

## Original v1 scripts

The original implementation used Bash/PowerShell and a direnv helper library.
Keep copies of your configuration before replacing it. Install the current
binary from the intended channel and ensure `command -v bwenv` resolves it.
Reinitialize each project using `bwenv init` and select its existing source.
Remove obsolete script/helper integration from the RC only after the new workflow
is verified; keep unrelated direnv configuration intact.

Use [installation](../INSTALL.md) for source/release instructions and
[troubleshooting](troubleshooting.md) if old wrappers or multiple binaries are
still taking precedence. Do not evaluate old help/setup text as shell code.
