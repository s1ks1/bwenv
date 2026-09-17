# Performance baseline

The reference baseline is v2.1.0. `bwenv benchmark` follows the current
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

The deterministic fake Bitwarden CLI integration test models one folder with
one item. A non-interactive export requires **four** provider invocations:

1. `IsAuthenticated`: list folders.
2. `Authenticate`: validate the same session by listing folders again.
3. `ListFolders`: resolve the folder name.
4. `GetSecrets`: list items in the folder.

Selected items add one `bw get item` invocation per item. These counts are
asserted by tests and are the baseline for the v2.3.0 fast path. Timing on a
fake CLI is not a useful measure of real vault latency; real measurements
require an authenticated development machine and are intentionally not
invented here.

The fake 1Password scenario currently uses five CLI invocations: two session
checks, one vault list, one item list, and one item detail fetch.

## Future comparison

| Version | Scenario | Median warm time | Provider processes |
|---|---|---:|---:|
| v2.1.0 | Bitwarden, full folder | Not measured with a real vault | 4 (fake CLI) |
| v2.3.0 | Bitwarden, full folder | Pending | Target: 1 |

The CI gate should assert process counts and output safety. Wall-clock timing
is recorded for local comparison, not used as a pass/fail threshold.
