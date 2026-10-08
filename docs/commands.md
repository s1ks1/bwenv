# Command reference

[Documentation home](../README.md) · [Getting started](getting-started.md) · [Automation](automation.md)

Syntax: `bwenv COMMAND [OPTIONS]`. Run `bwenv help` or `bwenv examples` for terminal
help. Flags go after the command; unexpected flags/arguments return an error.
`bwenv COMMAND --help` displays the general help. There is no global verbosity
flag, shell-persisted session store or daemon in the current implementation.

## User commands

| Command | Behavior | Options |
| --- | --- | --- |
| `init` | Interactive provider/folder/item setup and hook installation | `--activation shell\|direnv\|mise` |
| `login` | Authenticate, enable and load the nearest project through the wrapper | None |
| `allow` | Approve, authenticate if needed and load through the wrapper | None |
| `disallow` | Disable the nearest project and restore/clear its managed variables | None |
| `remove` | Remove canonical config and backend's generated artifacts; restore/clear variables | None |
| `refresh` | Explicit provider sync/reload; preserves disabled projects | `--shell bash\|zsh\|fish` for shell-code output |
| `lock` / `logout` | Lock/sign out available providers and clear invoking-shell credentials through wrapper | `--shell bash\|zsh\|fish` |
| `status` | Project/provider/session/preference overview, with masked credentials | None |
| `doctor` | Safe setup checks with suggested fixes; non-zero when required checks fail | None |
| `config` | Edit and save user preferences | None |
| `migrate` | Convert a supported legacy generated `.envrc` to canonical metadata | `--dry-run` |
| `benchmark` | Report provider timings, subprocess counts and variable count, without values | Provider selection flags below |
| `examples` | Terminal examples | None |
| `version` | Application version | Also `--version` or `-v` |
| `help` | General help | Also `--help`, `-h`, or no command |

`remove` leaves shared shell RC integration installed. For mise it removes its
generated source script/block while preserving unrelated settings. None of these
commands deletes your password manager items. Lock cleanup applies even if a
provider lock fails, but the wrapper preserves the failure exit status.

## Export

Through the installed wrapper, `bwenv export` applies its output to the current
shell. The raw executable prints shell assignments containing secret values.
To obtain raw output intentionally, use `command bwenv export` or an absolute
binary path. This distinction matters for pipelines and scripts.

```sh
bwenv export --project .
bwenv export --provider bitwarden --folder bwenv-demo
bwenv export --provider bitwarden --folder bwenv-demo --folder-id REAL_FOLDER_ID --items REAL_ITEM_ID
```

Replace `REAL_*` placeholders with actual IDs. Export requires existing provider
authentication; it does not prompt or sync.

| Flag | Description |
| --- | --- |
| `--project PATH` | Load `.bwenv.toml` from this directory or file; exclusive with selection flags |
| `--provider SLUG` | `bitwarden` or `1password`; required for direct selection |
| `--folder NAME` | Folder/vault name; required for direct selection even with a folder ID |
| `--folder-id ID` | Avoid folder lookup using a known stable ID |
| `--items ID1,ID2` | Restrict to these item IDs; omitted means all items |
| `--quiet` | Suppress automatic export messages; used by generated hooks |

Disabled project exports fail explicitly; quiet disabled exports succeed with no
assignments. Output dialect follows the shell environment; generated mise scripts
use Bash. For raw Fish output, set `SHELL` to your Fish executable if the inherited
value does not match the shell evaluating it.

## Benchmark

Inside an enabled project:

```sh
bwenv benchmark
```

Or use explicit selection:

```sh
bwenv benchmark --provider bitwarden --folder bwenv-demo --folder-id REAL_FOLDER_ID
```

Benchmark supports `--provider`, `--folder`, `--folder-id`, `--items`. It does not
support `--project` or `--quiet`. For reproducible before/after runs, use the
[performance measurement protocol](performance.md). Fake-provider measurements
are not real-vault latency results.

## Integration commands

These are primarily used by shell integrations. Prefer `login`, `allow` and
`disallow` for daily use.

| Command | Contract |
| --- | --- |
| `hook [bash\|zsh\|fish]` | Print native hook code; auto-detect shell if omitted |
| `activate [--shell SHELL]` | Activate nearest enabled project; native emits assignments, other backends approve |
| `deactivate [--shell SHELL]` | Restore native saved state when present; otherwise revoke/clear nearest project |
| `root [--shell-only] [--fingerprint]` | Print nearest canonical root or an opaque configuration fingerprint |
| `login-hint` | Print a concise instruction to log in; parent-shell integration uses it |

## Aliases

| Alias | Canonical command |
| --- | --- |
| `auth` | `login` |
| `load` | `export` |
| `clean` | `remove` |
| `test` | `status` (not the Go test suite) |
| `lock` | `logout` |
| `deny` | `disallow` |
| `settings` | `config` |

## Output and exit status

Shell-changing commands reserve raw stdout for valid assignments/unsets;
human messages go to stderr. Help/version/status/benchmark have their own human
formats and must not be evaluated as shell code.

| Exit code | Meaning |
| --- | --- |
| `0` | Success; may produce no output for a quiet disabled project |
| `1` | Invalid syntax, setup/provider failure or another command error |
| `2` | `export --quiet` requires authentication; hooks use this to request a login hint |

Do not assume `eval "$(command bwenv export ...)"` propagates an export failure.
For automation, check the command status before evaluating output; see the
[checked script recipe](automation.md#bashzsh-scripts).
