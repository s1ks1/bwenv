# Changelog

## Unreleased (planned v2.2.0)

### Added

- `bwenv benchmark` reports provider stages, total time, process count and
  variable count without printing secret values.
- A shared, injectable process runner for Bitwarden and 1Password commands.
- A deterministic fake provider CLI for integration tests, including expired
  sessions, malformed responses, provider errors and simulated latency.
- CI builds, tests and vets on Linux, macOS and Windows; Linux also checks
  formatting, race conditions, static analysis and known vulnerabilities.
- A performance baseline and measurement instructions in `docs/performance.md`.

### Notes

- This is a development milestone. No release tag has been created.
- The provider export behavior is unchanged; the next milestone optimizes it.
