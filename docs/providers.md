# Provider implementation

The `provider.Provider` interface supplies identity and CLI availability. Optional
interfaces supply authentication, folder/item listing, secret fetching, syncing
and locking: `Authenticator`, `FolderLister`, `SecretFetcher`, `Syncer`, `Locker`.
The interactive setup currently needs folder/item listing and authentication.

To add a provider:

1. Implement it in an isolated subpackage under `internal/provider`.
2. Register it with `provider.Register` and include its package in `provider/all`.
3. Retrieve key/value pairs using `SecretFetcher`; preserve selected item IDs.
4. Accept an injected `process.Runner` and expose `WithRunner` for isolated tests.
5. Wrap the existing typed errors with `%w`; callers use `errors.Is` rather than
   parsing CLI messages. Cover unavailable, unauthenticated, expired, malformed,
   not-found and timeout cases that the provider supports.
   Pass runner/decoder errors through `provider.SafeError` before rendering them;
   never display unwrapped causes or provider payloads. Interactive signin output
   must not be forwarded into shell-evaluated stdout.
6. Add fake-runner/parser tests. Assert subprocess counts for the warm path and
   prove failures do not reveal session tokens or raw provider payloads.

`AuthenticateNonInteractive` must never prompt or perform an implicit sync.
Session tokens go through child environment or supported input channels, not argv.
Do not import UI code into providers or add provider logic to activation scripts.
Use [security.md](security.md) as the baseline for secret handling.
