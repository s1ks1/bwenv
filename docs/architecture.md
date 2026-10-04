# Architecture

bwenv resolves secret references from a project and emits shell assignments.
Human messages go to stderr; the wrapper evaluates successful stdout in the parent
shell. Project files contain references, never credentials or decrypted values.

| Package | Responsibility |
| --- | --- |
| `cmd/bwenv`, root `main.go` | Thin executable entry points; root preserves existing `go install` paths |
| `internal/cli` | Syntax validation and aliases; no provider calls or TUI flows |
| `internal/app` | Typed command requests and application workflow coordination |
| `internal/buildinfo` | Shared release/module/VCS version resolution |
| `internal/export` | Activation/login/refresh orchestration, shell rendering and variable-name metadata |
| `internal/project` | Validated, versioned `.bwenv.toml` and project-root discovery |
| `internal/activation` | Activator registry and canonical source resolver |
| `internal/activation/{shell,direnv,mise}` | Hook artifacts, installation, approval and reload controls |
| `internal/provider` and provider subpackages | Capability interfaces, typed errors and provider-specific retrieval |
| `internal/process` | Injected provider subprocess execution, cancellation and instrumentation |
| `internal/session` | Provider reauthentication; tokens remain runtime state |
| `internal/ui`, `internal/output` | Interactive setup/preferences and terminal messages |
| `internal/diagnostics`, `internal/benchmark` | Measurements and safe reports |

The normal export path does not prompt or sync. A stable FolderID avoids folder
lookup; selected Bitwarden items use one list operation. Login can authenticate
interactively, preserves selection, and emits the session for the shell wrapper.
Direnv approval fails before any result is emitted.

Shell and mise share `activation.ResolveFromProjectConfig`; adapters do not fetch
provider data. Legacy `.envrc` parsing remains in the direnv adapter. Mise scripts
export into memory and keep automatic loads quiet; a parent-shell notice handles
expired authentication without repeating on every prompt.

Both entry points use the same CLI and application layers. Only entry points exit
the process; workflow handlers return a status, including the quiet-hook
authentication status. Release builds target `cmd/bwenv`; root builds remain
compatible with existing installation instructions.

The optional local agent and memory-only warm cache belong to PER-22/PER-23. See the
[workflow review](workflow-review.md) for current behavioral gaps and delivery order.
