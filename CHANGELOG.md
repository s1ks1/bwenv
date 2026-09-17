# Changelog

## Unreleased (planned v2.3.0)

### Performance

- Generated `.envrc` files now persist `--folder-id` and use it as the primary
  lookup key while older files keep the folder-name fallback.
- Non-interactive export no longer performs authentication preflight checks or
  Bitwarden syncs before fetching secrets.
- Bitwarden selected-item export fetches the folder item list once and filters
  selected IDs locally instead of starting one CLI process per item.
- Provider CLI commands now have a five-second non-interactive timeout.
- `.bwenv_vars` is not rewritten when the exported variable names are
  unchanged.

### Notes

- This is the v2.3.0 development line. Real-vault latency comparison remains
  local-only and is not used as a CI gate.

### Added

- `bwenv benchmark` reports provider stages, total time, process count and
  variable count without printing secret values.
- A shared, injectable process runner for Bitwarden and 1Password commands.
- A deterministic fake provider CLI for integration tests, including expired
  sessions, malformed responses, provider errors and simulated latency.
- CI builds, tests and vets on Linux, macOS and Windows; Linux also checks
  formatting, race conditions, static analysis and known vulnerabilities.
- A performance baseline and measurement instructions in `docs/performance.md`.

## v2.2.0 - 2026-09-17

### Added

- Phase 0 diagnostics, benchmark reporting, fake provider CLI coverage and
  cross-platform CI guardrails.
- Performance baseline documentation in `docs/performance.md`.
