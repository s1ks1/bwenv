# Changelog

## Unreleased

### Added

- `bwenv doctor` reports actionable setup checks with output safe to share in
  issue reports, including FolderID fast-path and `.envrc` permission checks.
- `bwenv refresh` syncs Bitwarden on demand and triggers a direnv reload;
  1Password refreshes through its normal provider request without a sync step.
- Removed the obsolete Auto Sync preference; `bwenv export` remains sync-free.
- New projects with a stable folder ID store versioned, secret-free references in
  `.bwenv.toml`; generated `.envrc` files load them with `bwenv export --project`.
- Existing `.envrc` projects and direct provider/folder export flags remain
  supported. See `docs/migration.md` for the metadata format and compatibility.
- `bwenv migrate --dry-run` previews legacy project changes; `bwenv migrate`
  writes canonical metadata and keeps a permission-restricted `.envrc` backup.
- Migration preserves existing `BW_SESSION` behavior until v3 runtime session
  management is available, and refuses custom shell code it cannot preserve.

## v2.3.0 - 2026-09-17

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

- On the reference macOS machine, a warm Bitwarden export improved from
  7182.4 ms with four provider processes to 3112.9 ms with one process.
- The measured improvement is approximately 56.7%; wall-clock timing remains
  a local observation and is not used as a CI gate.

## v2.2.0 - 2026-09-17

### Added

- Phase 0 diagnostics, benchmark reporting, fake provider CLI coverage and
  cross-platform CI guardrails.
- Performance baseline documentation in `docs/performance.md`.
- `bwenv benchmark` reports provider stages, total time, process count and
  variable count without printing secret values.
- A shared, injectable process runner for Bitwarden and 1Password commands.
- CI builds, tests and vets on Linux, macOS and Windows; Linux also checks
  formatting, race conditions, static analysis and known vulnerabilities.
