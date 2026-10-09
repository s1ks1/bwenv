# Troubleshooting

[Documentation home](../README.md) · [Installation](../INSTALL.md) · [Activation](activation.md)

Start in the affected project and run:

```sh
bwenv --version
bwenv doctor
```

Doctor is designed for shareable diagnosis and excludes secret values, tokens,
provider payloads and project paths. `status` gives more project context; review
metadata before sharing it. Never attach raw exports, a full environment, shell
traces or screenshots containing credentials.

## Find the symptom

| Symptom | First check |
| --- | --- |
| `bwenv: command not found` | [PATH and multiple installations](#path-and-multiple-installations) |
| `'bw' CLI is not installed` or missing `op` | Provider CLI available in the same shell |
| `bwenv login` asks you to type eval | [Wrapper not active](#login-does-not-update-the-current-shell) |
| Login hint on project entry | [Session missing or expired](#session-missing-or-expired) |
| No hint and no variables | [No variables loaded](#no-variables-loaded) |
| Box borders interpreted as commands at startup | [Invalid eval in RC](#startup-errors-from-eval) |
| Repeated summaries after unrelated commands | [Duplicate integrations or old mise script](#repeated-messages) |
| Mise hangs until Ctrl+C | [Node shims and mise source configuration](#mise-hangs-or-keeps-reloading) |
| TOML error or backend mismatch | [Project configuration](#invalid-or-stale-project-configuration) |

## PATH and multiple installations

```sh
command -v bwenv
command -v bw
command -v op
bwenv --version
```

Only your chosen provider is required. In Bash/Zsh, `type -a bwenv` also reveals
whether a wrapper and multiple binary installations coexist. Update the intended
binary, ensure its directory is on PATH, and open a new terminal. A v2 binary
cannot provide all v3 commands. Inside WSL use the Linux provider/binary/session;
Windows and WSL environments are separate.

## Login does not update the current shell

`init` can modify your RC file, but that does not reload the current shell.
Source the exact file shown by setup, or open a new terminal. Run `bwenv login`
inside the project again. Bash/Zsh `type bwenv` should identify a function;
Fish `functions bwenv` should show the installed wrapper.

For a one-time Bash/Zsh fallback when the wrapper is unavailable:

```sh
if exports="$(command bwenv login)"; then
  eval "$exports"
  unset exports
fi
```

Use the installed wrapper for daily work. Do not evaluate help/status/version
text. With 1Password without desktop integration, establish a valid provider CLI
session first; bwenv does not import token-bearing signin stdout.

## Session missing or expired

Run `bwenv login` through the active wrapper. Re-entering a project in the same
shell reuses its valid Bitwarden session; a new terminal has its own environment.
Persistent secret-free preferences do not imply persistent authentication.

For Bitwarden, if the account was never established, run `bw login` first. If
1Password cannot authenticate, check desktop CLI integration and account access,
or the securely supplied service-account/session credentials. Repeated failure
can indicate provider connectivity or permissions; it is not fixed by repeated eval.

## No variables loaded

1. Run `bwenv doctor` inside the affected directory.
2. Inspect `.bwenv.toml`: provider, folder/vault ID, selected item IDs and backend.
3. If disabled, use `bwenv allow` or `bwenv login` intentionally to enable it.
4. Confirm fields are supported: Bitwarden custom fields; 1Password labeled
   non-empty fields except notes/OTP. See [field rules](getting-started.md#field-naming-rules).
5. Ensure the external direnv/mise hook is loaded and project trust is granted.
6. Run `bwenv refresh` if provider data changed after a Bitwarden sync.
7. Verify presence of one known test variable without printing values.

An empty item list means all items. A stale explicit item list may omit newly
added items. Re-run init to select them or intentionally update canonical metadata.
Native activation follows the nearest canonical project. `export --project .`
requires a file in that specific directory; use the root path from a nested folder.

## Startup errors from eval

Messages such as `command not found: ╭`, `command not found: Setup`, or an error
about `init` mean formatted terminal text was passed to eval in the RC file.
Inspect the relevant lines in `~/.zshrc`, `~/.bashrc`, `~/.bash_profile` or Fish config.
Remove an accidental expression such as `eval "$(bwenv init)"` or one evaluating help.

Native Zsh integration can be loaded with:

```sh
eval "$(bwenv hook zsh)"
```

Use the [correct command for your chosen backend/shell](activation.md). Keep one
copy of each applicable integration. Re-run init with the current binary to
upgrade recognized generated blocks, then open a new terminal. Custom RC code
still requires manual review; keep a backup before editing it.

## Repeated messages

- Keep a single installed wrapper and matching hook block in the RC file.
- Update the binary actually resolved by PATH.
- Regenerate old artifacts with `bwenv init --activation YOUR_BACKEND`, preserving
  the intended provider/folder/items. Review custom scripts before regeneration.
- Source the changed RC file or open a new terminal.
- Turn off **Show Export Summary** for quieter explicit operations if desired.

Mise automatic loads should be silent. A login hint once per project entry is
expected when authentication is unavailable; a success summary after every
unrelated command suggests an older source script or another integration printing it.

## Mise hangs or keeps reloading

Check `mise.toml` and the source script. The bwenv source requires:

```toml
[env]
_.source = { path = ".bwenv.mise.sh", tools = true }
```

With Node-based `bw`, leaving out `tools = true` can route provider execution back
through mise shims while mise is already evaluating the environment. Regenerate
with `bwenv init --activation mise` using the current binary, review the files,
then run `mise trust` and `bwenv login` in a shell with the mise hook loaded.

If another script owns `env._.source`, do not overwrite it blindly. Integrate the
sources manually and keep bwenv's active-tool requirement. On a persistent
failure use explicit `bwenv doctor`, verify Node/`bw` availability, and report the
mise/Node/provider versions. A Ctrl+C cancellation alone does not identify the cause.

## Invalid or stale project configuration

Unknown TOML fields and unsupported schema versions are errors. An invalid
`.bwenv.toml` does not fall back to a legacy `.envrc`. Check against the
[configuration reference](configuration.md). Prefer reinitializing after keeping
a copy of reviewed metadata; changing `mode` alone does not install another hook.

For an older generated `.envrc`:

```sh
bwenv migrate --dry-run
bwenv migrate
```

Migration refuses unsupported custom commands. Review them manually rather than
forcing a conversion. See [migration](migration.md) for backup handling.

## Lock and cleanup surprises

Lock clears the invoking shell; other terminals and running children retain their
own environment copies. Removing a local service-account token does not revoke
it remotely. With `OP_SERVICE_ACCOUNT_TOKEN`, logout removes that token through
the shell wrapper instead of running `op signout`; supplying the token again is
required to reuse the service account. If your shell startup config supplies it,+a new terminal will receive it again. A provider lock error still produces shell cleanup through the wrapper,
but returns a failure so you can investigate the provider operation.

`disallow` persists disablement for native/mise projects. `remove` deletes the
project's bwenv artifacts, not the vault's data or shared RC hook. If these commands
print unset code instead of applying it, activate the wrapper first.

## Reporting a bug

Include the bwenv version, OS/architecture, shell and chosen hook, provider CLI
version, shareable doctor output, the exact non-sensitive command and expected
versus actual behavior. Reproduce with a dedicated fake/demo value if possible.
Review IDs/names in metadata before attaching it. Follow [SECURITY.md](../SECURITY.md)
for vulnerabilities instead of posting credentials publicly.
