# Performance baseline

The reference baseline is v2.2.0. `bwenv benchmark` follows the current
non-interactive provider path and reports stage durations and the number of
provider CLI processes. It never prints secret values or command arguments.

## Reproduce locally

Run from a configured project directory with an active provider session:

```bash
bwenv benchmark
```

Or provide the same arguments used by `bwenv export`:

```bash
bwenv benchmark --provider bitwarden --folder Production
```

Record the OS, architecture, provider CLI version, number of selected items,
cold and warm durations, and process count for each measurement. Run the warm
case several times and report the median. Do not include secret values or
session tokens in a report.

## Verified structural baseline

The v2.2.0 deterministic fake Bitwarden path required **four** provider
invocations:

1. `IsAuthenticated`: list folders.
2. `Authenticate`: validate the same session by listing folders again.
3. `ListFolders`: resolve the folder name.
4. `GetSecrets`: list items in the folder.

The v2.3.0 fast path persists the folder ID and uses one `bw list items`
invocation for the full folder or for selected items, then filters locally.
Non-interactive export does not run `bw sync` or a separate session probe.
These process counts and the absence of secret values in benchmark output are
asserted by tests.

Older `.envrc` files without `--folder-id` remain supported and use one extra
folder-list operation. The fake 1Password fast path uses one item-list and one
item-detail process for a one-item vault.

## v2.3.0 comparison

The following warm run was measured on the reference macOS machine with two
exported variables. It is a local observation, not a CI gate.

| Version | Scenario | Warm time | Provider processes |
|---|---|---:|---:|
| v2.2.0 | Bitwarden, full folder | 7182.4 ms | 4 (legacy path) |
| v2.3.0 | Bitwarden, full folder | 3112.9 ms | 1 (FolderID path) |

The observed improvement is approximately 56.7%. Repeat the measurement several
times and report the median when comparing another machine or provider CLI
version.

The CI gate should assert process counts and output safety. Wall-clock timing
is recorded for local comparison, not used as a pass/fail threshold.
