# Contributing

Use the Go version declared in `go.mod`. The local loop is:

```sh
make build
make test
make lint
go test -race ./...
go vet ./...
```

Format changed Go files with `gofmt`; CI checks formatting. Local builds produce
`dist/bwenv`. Run `go test -v .` for shell/provider integration coverage; these tests
use fake credentials. Tests requiring unavailable optional tools may skip. Check
CI on Linux, macOS and Windows before release, rather than inferring platform
coverage from one machine.

Work on a focused branch and link the relevant Linear issue. Describe the concrete
before/after behavior in a PR and include validation and known limitations. Preserve
existing project modes and legacy migration behavior. Add a focused regression for
behavior changes, particularly shell evaluation, sessions and error handling.

Never include real secret values, session tokens, exported environments or provider
payloads in test fixtures, logs, issues or PRs. Keep generated assignments on stdout
and human messages on stderr. Do not add decrypted disk caches for performance.
See [architecture](docs/architecture.md), [providers](docs/providers.md), and
[security](docs/security.md) before changing the activation hot path.

Before a release, follow [release readiness](docs/release-readiness.md). Manual
workflow dispatches rehearse packaging; they do not publish or push tags.
