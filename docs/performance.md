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

## Comparison

| Version | Scenario | Warm time | Provider processes |
|---|---|---:|---:|
| v2.2.0 | Bitwarden, full folder | 7182.4 ms¹ | 4 (legacy path) |
| v2.3.0 | Bitwarden, full folder | 3112.9 ms¹ | 1 (FolderID path) |
| v2.4.0-dev | Bitwarden, full folder, `--folder` only (folder resolution) | 5133.0 ms² | 2 |
| v2.4.0-dev | Bitwarden, full folder, `--folder-id` (fast path) | 2705.5 ms² | 1 |

¹ Single run (historical, two exported variables); predates the median
protocol above.

² Median of five warm runs on 2026-09-27, macOS arm64 (Darwin arm64),
bw CLI 2026.9.0, folder `bvenv` (2 items, 4 variables), no `bw sync`.
Folder-resolution runs: 5027.5–5230.3 ms. Fast-path runs: 2699.0–2805.1 ms
after one discarded warm-up.

Relative to the v2.2.0 single-run baseline, the v2.4.0 fast-path median is
approximately 62.3% lower (7182.4 → 2705.5 ms). Resolving the folder name
instead of passing `--folder-id` costs roughly 2.4 s extra (one extra
`bw list folders` process), which is why generated `.envrc` files persist the
folder ID. All numbers are local observations, not CI gates.

The CI gate should assert process counts and output safety. Wall-clock timing
is recorded for local comparison, not used as a pass/fail threshold.
