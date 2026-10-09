# Changelog

## Unreleased

- Pass the selected vault ID when fetching 1Password items, fixing preview and export for service accounts with access to multiple vaults.

## v3.0.0 - 2026-10-08

- Publish the complete v3 user documentation, command reference, migration guide and accessible workflow demo.
- Retain v2.4.1 downloads and the v2 maintenance branch for possible compatibility patches.

- Harden CI and release preparation with pinned actions/tools, cross-platform checks, unpublished packaging rehearsals, SBOM/checksum verification and draft-only provenance configuration.
- Require exact checksums in both installers; fix Windows home-directory fixtures and validate Fish/mise Node workflows.
- Fix the Windows x/sys vulnerability; source builds now require Go 1.25+.

- Publish the threat model; isolate 1Password signin output, redact runner/decoder errors while preserving typed causes, and reject NUL values before shell output.

- Separate CLI syntax parsing, application workflows and UI rendering; release builds use a thin `cmd/bwenv` entry point while existing root installation paths remain supported.
- Reject unknown commands/options without shell-evaluated output; fix `--version` and `-v`.

- Persist native/mise project disablement until allow/login; directory exit remains temporary deactivation.
- Lock/logout restores or clears managed values and removes provider session variables from the invoking shell, including on provider failure. Automatic loads remain blocked until successful authentication; existing wrappers are upgraded in place.
- Keep mise variable metadata in the referenced project, normalize login-notice paths and upgrade generated scripts on refresh without touching custom scripts.

- Reload native shell secrets when the project config changes in the same directory, restoring variables from the previous selection before loading the new note.
- Fail allow/login before shell output when project approval fails; bound direnv allow/deny/reload and surface actionable failures.
- Resolve explicit project commands from subdirectories, share the canonical activation resolver, and verify malicious disk-derived unset names through real Bash eval.
- Remove plaintext mise env caching and clean legacy caches on installation/reload; a secure memory-only cache remains future agent work.
- Add current architecture, provider, security, contribution and workflow-review documentation.

- Mise uses quiet exports during automatic evaluation, avoiding startup warnings and repeated summaries on every prompt; explicit login/export commands retain diagnostics.

- Unified compact command and hook messages; reloads after login no longer repeat unchanged load summaries. Interactive failures have one owner, direnv status chatter is filtered, and logout no longer repeats its results.

- Mise sources secrets after tool activation (`tools = true`), avoiding recursion through Node shims. Locked vaults leave tool activation available for login, and subprocess pipe waits are bounded after cancellation.

- Existing shell wrappers are upgraded to evaluate `bwenv login` automatically; activation failures are concise and are not retried on every prompt until the session or project changes.

- New projects default to native shell activation; `bwenv config` saves a default choice of shell, direnv, or mise. Existing project modes remain authoritative.
- Init prints the selected backend and required shell setup steps. Native and mise backends remain experimental.
- Fixed hook argument parsing, project switching, environment restoration, activation retries, and mise registration, canonical metadata, relative paths and config preservation.
- Validated cached unset names and runtime state, tightened project metadata to mode 0600, and surfaced approval failures.

## v2.4.1 - 2026-09-27

### Fixed

- `bwenv export --project .` (the command every generated `.envrc` runs) failed
  with `read .: is a directory` after `bwenv init`; directory project paths now
  resolve to `<dir>/.bwenv.toml`.

## v2.4.0 - 2026-09-27

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

### Fixed

- Migrated the module path to `github.com/s1ks1/bwenv/v2` so
  `go install github.com/s1ks1/bwenv/v2@latest` works from the next tagged
  release.
- Plain `go build` / `go install` now reports a real version derived from build
  info (module version or VCS revision) instead of the hardcoded `v2.2.0-dev`.

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
